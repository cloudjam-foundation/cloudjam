import base64
import concurrent.futures
import datetime
import fnmatch
import hmac
import json
import math
import os
import time
import urllib.error
import urllib.parse
import urllib.request

import boto3
from botocore.config import Config
from botocore.exceptions import BotoCoreError, ClientError

REGION = "eu-central-1"
PREFIX = "cloudjam-gridline-assessor-"
CLIENTS = {}
CACHE = None
LOADED = 0


def client(service):
    if service not in CLIENTS:
        CLIENTS[service] = boto3.client(
            service, region_name=REGION, config=Config(connect_timeout=2, read_timeout=4, retries={"max_attempts": 1})
        )
    return CLIENTS[service]


def fetch(service, operation, key=None, **args):
    connection = client(service)
    if key and connection.can_paginate(operation):
        return [item for page in connection.get_paginator(operation).paginate(**args) for item in page.get(key, [])]
    response = getattr(connection, operation)(**args)
    return response.get(key, []) if key else response


def optional(service, operation, **args):
    try:
        return fetch(service, operation, **args)
    except ClientError as error:
        if error.response["Error"]["Code"] in (
            "NoSuchLifecycleConfiguration",
            "NoSuchBucketPolicy",
            "NoSuchPublicAccessBlockConfiguration",
            "ServerSideEncryptionConfigurationNotFoundError",
            "NoSuchKey",
        ):
            return {}
        raise


def inventory(section):
    if section == "network":
        return {
            key: fetch("ec2", operation, key)
            for key, operation in (
                ("Vpcs", "describe_vpcs"),
                ("Subnets", "describe_subnets"),
                ("TransitGateways", "describe_transit_gateways"),
                ("TransitGatewayAttachments", "describe_transit_gateway_attachments"),
                ("SecurityGroups", "describe_security_groups"),
                ("VpcPeeringConnections", "describe_vpc_peering_connections"),
                ("VpcEndpoints", "describe_vpc_endpoints"),
            )
        }
    if section == "database":
        return {
            "clusters": fetch("rds", "describe_db_clusters", "DBClusters"),
            "instances": fetch("rds", "describe_db_instances", "DBInstances"),
            "subnets": fetch("rds", "describe_db_subnet_groups", "DBSubnetGroups"),
        }
    if section == "containers":
        clusters, services, tasks = [], [], {}
        for arn in fetch("ecs", "list_clusters", "clusterArns"):
            clusters.extend(fetch("ecs", "describe_clusters", "clusters", clusters=[arn]))
            service_arns = fetch("ecs", "list_services", "serviceArns", cluster=arn)
            for offset in range(0, len(service_arns), 10):
                services.extend(
                    fetch(
                        "ecs", "describe_services", "services", cluster=arn, services=service_arns[offset : offset + 10]
                    )
                )
        for service in services:
            arn = service["taskDefinition"]
            if arn not in tasks:
                tasks[arn] = fetch("ecs", "describe_task_definition", taskDefinition=arn)["taskDefinition"]
        return {"clusters": clusters, "services": services, "tasks": tasks}
    if section == "edge":
        distributions = fetch("cloudfront", "list_distributions").get("DistributionList", {}).get("Items", [])
        policies = {}
        for distribution in distributions:
            behavior = distribution.get("DefaultCacheBehavior", {})
            if behavior.get("CachePolicyId"):
                policies[behavior["CachePolicyId"]] = fetch(
                    "cloudfront", "get_cache_policy", Id=behavior["CachePolicyId"]
                )["CachePolicy"]["CachePolicyConfig"]
        return {
            "distributions": distributions,
            "policies": policies,
        }
    if section == "api":
        integrations = []
        for api in fetch("apigatewayv2", "get_apis", "Items"):
            if api.get("ProtocolType") != "HTTP":
                continue
            targets = {
                item["IntegrationId"]: item
                for item in fetch("apigatewayv2", "get_integrations", "Items", ApiId=api["ApiId"])
            }
            for route in fetch("apigatewayv2", "get_routes", "Items", ApiId=api["ApiId"]):
                if route.get("RouteKey") in ("POST /transponder", "ANY /transponder", "ANY /{proxy+}", "$default"):
                    target = targets.get(route.get("Target", "").split("/")[-1], {})
                    if target.get("IntegrationType") == "AWS_PROXY":
                        integrations.append(target.get("IntegrationUri", ""))
        for api in fetch("apigateway", "get_rest_apis", "items"):
            for resource in fetch("apigateway", "get_resources", "items", restApiId=api["id"]):
                if resource.get("path") not in ("/transponder", "/{proxy+}"):
                    continue
                for method in ("POST", "ANY"):
                    if method in resource.get("resourceMethods", {}):
                        target = fetch(
                            "apigateway",
                            "get_integration",
                            restApiId=api["id"],
                            resourceId=resource["id"],
                            httpMethod=method,
                        )
                        if target.get("type") == "AWS_PROXY":
                            integrations.append(target.get("uri", ""))
        return integrations
    if section == "functions":
        return [
            function
            for function in fetch("lambda", "list_functions", "Functions")
            if not function["FunctionName"].startswith(PREFIX)
        ]
    if section == "identity":
        return [
            {"pool": pool, "groups": fetch("cognito-idp", "list_groups", "Groups", UserPoolId=pool["Id"])}
            for pool in fetch("cognito-idp", "list_user_pools", "UserPools", MaxResults=60)
        ]
    if section == "configuration":
        applications = []
        for application in fetch("appconfig", "list_applications", "Items"):
            app_id = application["Id"]
            profiles = []
            for profile in fetch("appconfig", "list_configuration_profiles", "Items", ApplicationId=app_id):
                profile = fetch(
                    "appconfig", "get_configuration_profile", ApplicationId=app_id, ConfigurationProfileId=profile["Id"]
                )
                profile["documents"] = []
                if profile.get("LocationUri") == "hosted":
                    for version in fetch(
                        "appconfig",
                        "list_hosted_configuration_versions",
                        "Items",
                        ApplicationId=app_id,
                        ConfigurationProfileId=profile["Id"],
                    ):
                        content = fetch(
                            "appconfig",
                            "get_hosted_configuration_version",
                            ApplicationId=app_id,
                            ConfigurationProfileId=profile["Id"],
                            VersionNumber=version["VersionNumber"],
                        )
                        try:
                            profile["documents"].append(json.loads(content["Content"].read()))
                        except (ValueError, UnicodeDecodeError):
                            pass
                profiles.append(profile)
            environments = fetch("appconfig", "list_environments", "Items", ApplicationId=app_id)
            deployments = [
                deployment
                for environment in environments
                for deployment in fetch(
                    "appconfig", "list_deployments", "Items", ApplicationId=app_id, EnvironmentId=environment["Id"]
                )
            ]
            applications.append({"application": application, "profiles": profiles, "deployments": deployments})
        return applications
    if section == "workflow":
        rules = []
        for bus in fetch("events", "list_event_buses", "EventBuses"):
            for rule in fetch("events", "list_rules", "Rules", EventBusName=bus["Name"]):
                rule["targets"] = fetch(
                    "events", "list_targets_by_rule", "Targets", Rule=rule["Name"], EventBusName=bus["Name"]
                )
                rules.append(rule)
        machines = [
            fetch("stepfunctions", "describe_state_machine", stateMachineArn=machine["stateMachineArn"])
            for machine in fetch("stepfunctions", "list_state_machines", "stateMachines")
        ]
        return {"rules": rules, "machines": machines}
    if section == "stream":
        return [
            fetch("firehose", "describe_delivery_stream", DeliveryStreamName=name)["DeliveryStreamDescription"]
            for name in fetch("firehose", "list_delivery_streams", "DeliveryStreamNames")
        ]
    if section == "messaging":
        queues = []
        for url in fetch("sqs", "list_queues", "QueueUrls"):
            queues.append(
                {
                    "url": url,
                    "attributes": fetch("sqs", "get_queue_attributes", QueueUrl=url, AttributeNames=["All"])[
                        "Attributes"
                    ],
                }
            )
        return {"queues": queues, "subscriptions": fetch("sns", "list_subscriptions", "Subscriptions")}
    if section == "pipeline":
        pipelines, projects = [], {}
        for item in fetch("codepipeline", "list_pipelines", "pipelines"):
            pipeline = fetch("codepipeline", "get_pipeline", name=item["name"])["pipeline"]
            pipeline["state"] = fetch("codepipeline", "get_pipeline_state", name=item["name"])
            pipelines.append(pipeline)
            for stage in pipeline["stages"]:
                for action in stage["actions"]:
                    if action["actionTypeId"]["provider"] == "CodeBuild":
                        name = action["configuration"].get("ProjectName")
                        for project in fetch("codebuild", "batch_get_projects", "projects", names=[name]):
                            projects[name] = project
        return {"pipelines": pipelines, "projects": projects}
    if section == "scaling":
        return {
            "targets": fetch(
                "application-autoscaling", "describe_scalable_targets", "ScalableTargets", ServiceNamespace="ecs"
            ),
            "policies": fetch(
                "application-autoscaling", "describe_scaling_policies", "ScalingPolicies", ServiceNamespace="ecs"
            ),
        }
    if section == "banned":
        return {
            "zones": fetch("route53", "list_hosted_zones", "HostedZones"),
            "caches": fetch("elasticache", "describe_cache_clusters", "CacheClusters"),
            "serverless_caches": fetch("elasticache", "describe_serverless_caches", "ServerlessCaches"),
            "networks": fetch("vpc-lattice", "list_service_networks", "items"),
            "services": fetch("vpc-lattice", "list_services", "items"),
        }
    if section == "logs":
        return fetch("logs", "describe_log_groups", "logGroups")
    raise ValueError(section)


def snapshot():
    global CACHE, LOADED
    if CACHE is not None and time.time() - LOADED < 120:
        return CACHE
    sections = [
        "network",
        "database",
        "containers",
        "edge",
        "api",
        "functions",
        "identity",
        "configuration",
        "workflow",
        "stream",
        "messaging",
        "pipeline",
        "scaling",
        "banned",
        "logs",
    ]
    data, errors = {}, {}
    for service in (
        "ec2",
        "rds",
        "ecs",
        "cloudfront",
        "apigatewayv2",
        "apigateway",
        "lambda",
        "cognito-idp",
        "appconfig",
        "events",
        "stepfunctions",
        "firehose",
        "sns",
        "sqs",
        "codepipeline",
        "codebuild",
        "application-autoscaling",
        "route53",
        "elasticache",
        "vpc-lattice",
        "logs",
        "s3",
        "iam",
    ):
        client(service)
    with concurrent.futures.ThreadPoolExecutor(max_workers=12) as executor:
        futures = {executor.submit(inventory, section): section for section in sections}
        for future in concurrent.futures.as_completed(futures):
            section = futures[future]
            try:
                data[section] = future.result()
            except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
                errors[section] = str(error)[:240]
    data["errors"] = errors
    data["buckets"], data["policies"] = {}, {}
    bucket_names = set()
    for distribution in data.get("edge", {}).get("distributions", []):
        for origin in distribution.get("Origins", {}).get("Items", []):
            if ".s3" in origin.get("DomainName", ""):
                bucket_names.add(origin["DomainName"].split(".s3")[0])
    for stream in data.get("stream", []):
        for destination in stream.get("Destinations", []):
            s3 = destination.get("ExtendedS3DestinationDescription", destination.get("S3DestinationDescription", {}))
            if s3.get("BucketARN"):
                bucket_names.add(s3["BucketARN"].split(":::")[-1])
    for function in data.get("functions", []):
        bucket = function.get("Environment", {}).get("Variables", {}).get("TELEMETRY_BUCKET")
        if bucket:
            bucket_names.add(bucket)
    for name in bucket_names:
        try:
            data["buckets"][name] = {
                "lifecycle": optional("s3", "get_bucket_lifecycle_configuration", Bucket=name).get("Rules", []),
                "public": optional("s3", "get_public_access_block", Bucket=name).get(
                    "PublicAccessBlockConfiguration", {}
                ),
                "region": fetch("s3", "get_bucket_location", Bucket=name).get("LocationConstraint") or "us-east-1",
            }
        except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
            errors["bucket:" + name] = str(error)[:240]
    roles = {
        task["taskRoleArn"] for task in data.get("containers", {}).get("tasks", {}).values() if task.get("taskRoleArn")
    }
    roles.update(function["Role"] for function in data.get("functions", []))
    for arn in roles:
        name = arn.split("/")[-1]
        try:
            documents = [
                fetch("iam", "get_role_policy", RoleName=name, PolicyName=policy)["PolicyDocument"]
                for policy in fetch("iam", "list_role_policies", "PolicyNames", RoleName=name)
            ]
            for attached in fetch("iam", "list_attached_role_policies", "AttachedPolicies", RoleName=name):
                policy = fetch("iam", "get_policy", PolicyArn=attached["PolicyArn"])["Policy"]
                documents.append(
                    fetch(
                        "iam",
                        "get_policy_version",
                        PolicyArn=attached["PolicyArn"],
                        VersionId=policy["DefaultVersionId"],
                    )["PolicyVersion"]["Document"]
                )
            data["policies"][arn] = documents
        except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
            errors["role:" + arn] = str(error)[:240]
    CACHE, LOADED = data, time.time()
    return data


def request(base, method, path, body=None, group="spectator"):
    headers = {"Accept": "application/json", "X-CloudJam-Check": "architecture"}
    if group:
        headers.update({"X-Dev-Sub": "scoring-" + group, "X-Dev-Groups": group})
    if body is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(
        base.rstrip("/") + path,
        method=method,
        headers=headers,
        data=json.dumps(body).encode() if body is not None else None,
    )
    try:
        response = urllib.request.urlopen(req, timeout=3)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        payload = response.read(1024 * 1024)
        try:
            payload = json.loads(payload)
        except ValueError:
            payload = {}
        return response.status, payload


def advance(evidence):
    probe = evidence.get("probe") or {}
    if not evidence.get("active") or not evidence.get("endpoint"):
        return probe
    base, now = evidence["endpoint"], time.time()
    current = probe.get("current")
    if current and now - current["created"] > 300:
        probe["last"] = dict(current, finished=False, detail="Heat did not finish within five minutes")
        current = None
        probe.pop("current", None)
    try:
        if current is None:
            if evidence.get("remaining", 0) < 240 or now - probe.get("completed", 0) < 120:
                return probe
            status, heat = request(
                base,
                "POST",
                "/api/heats",
                {"track_id": "track-01", "label": "CloudJam architecture", "duration_seconds": 180},
                "director",
            )
            if status != 201 or not heat.get("heat_id"):
                probe["last"] = {"detail": "Architecture heat creation returned HTTP " + str(status)}
                return probe
            current = {
                "heat_id": heat["heat_id"],
                "created": now,
                "laps": 0,
                "batches": 0,
                "min_rejected": False,
                "max_rejected": False,
            }
            probe["current"] = current
            status, _ = request(base, "POST", "/api/heats/" + current["heat_id"] + "/start", {}, "director")
            current["started"] = status == 202
        status, heat = request(base, "GET", "/api/heats/" + current["heat_id"])
        if status != 200:
            current["detail"] = "Heat lookup returned HTTP " + str(status)
            return probe
        if heat.get("status") == "RUNNING":
            current["running"] = True
            started = heat.get("started_at")
            if started:
                current["running_at"] = datetime.datetime.fromisoformat(started.replace("Z", "+00:00")).timestamp()
            else:
                current.setdefault("running_at", now)
            elapsed = now - current["running_at"]
            if current["batches"] < 2 and elapsed >= 60 * (current["batches"] + 1):
                for transponder in ("T-0001", "T-0002", "T-0003"):
                    body = {
                        "transponder_id": transponder,
                        "track_id": "track-01",
                        "heat_id": current["heat_id"],
                        "line": "sf",
                        "ts": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    }
                    code, result = request(base, "POST", "/transponder", body, "")
                    if code == 202 and result.get("lap_number") and result.get("rules_version"):
                        current["laps"] += 1
                    else:
                        current["detail"] = "Valid crossing returned HTTP " + str(code)
                    if transponder == "T-0001" and code == 202:
                        body["ts"] = (
                            datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(milliseconds=1)
                        ).isoformat()
                        current["min_rejected"] = request(base, "POST", "/transponder", body, "")[0] == 422
                        body["ts"] = (
                            datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=1)
                        ).isoformat()
                        current["max_rejected"] = request(base, "POST", "/transponder", body, "")[0] == 422
                current["batches"] += 1
            code, board = request(base, "GET", "/api/heats/" + current["heat_id"] + "/leaderboard")
            current["standings"] = code == 200 and len(board.get("standings", [])) >= 3
        elif heat.get("status") in ("FINISHED", "FAILED"):
            current["finished"] = heat["status"] == "FINISHED" and bool(heat.get("finished_at"))
            probe["last"] = current
            probe["completed"] = now
            probe.pop("current", None)
    except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
        if current:
            current["detail"] = str(error)[:240]
        else:
            probe["last"] = {"detail": str(error)[:240]}
    return probe


def assess(data, evidence):
    results = {}
    errors = data.get("errors", {})

    def mark(identifier, passed, detail, *sections):
        unavailable = [section + ": " + errors[section] for section in sections if section in errors]
        results[identifier] = {
            "ok": bool(passed) and not unavailable,
            "detail": "; ".join(unavailable) if unavailable else detail,
        }

    def passed(*ids):
        return all(evidence.get("checks", {}).get(identifier, {}).get("ok", False) for identifier in ids)

    network = data.get("network", {})
    subnets = {subnet["SubnetId"]: subnet for subnet in network.get("Subnets", [])}
    tasks = data.get("containers", {}).get("tasks", {})
    services = data.get("containers", {}).get("services", [])
    apps = {
        name: [
            (service, tasks.get(service["taskDefinition"], {}))
            for service in services
            if any(
                name in (container.get("name", "") + " " + container.get("image", ""))
                for container in tasks.get(service["taskDefinition"], {}).get("containerDefinitions", [])
            )
        ]
        for name in ("race-api", "timing-engine")
    }
    application_services = {service["serviceArn"]: service for group in apps.values() for service, _ in group}.values()
    application_tasks = {task["taskDefinitionArn"]: task for group in apps.values() for _, task in group}.values()
    containers = [
        container
        for task in application_tasks
        for container in task.get("containerDefinitions", [])
        if any(name in (container.get("name", "") + " " + container.get("image", "")) for name in apps)
    ]
    track_vpcs = {
        subnets[subnet]["VpcId"]
        for service in application_services
        for subnet in service.get("networkConfiguration", {}).get("awsvpcConfiguration", {}).get("subnets", [])
        if subnet in subnets
    }
    clusters = [
        cluster
        for cluster in data.get("database", {}).get("clusters", [])
        if cluster.get("Engine") == "aurora-postgresql"
    ]
    db_groups = {group["DBSubnetGroupName"]: group for group in data.get("database", {}).get("subnets", [])}
    data_vpcs = {
        db_groups[cluster["DBSubnetGroup"]]["VpcId"]
        for cluster in clusters
        if cluster.get("DBSubnetGroup") in db_groups
    }
    instances = [
        instance
        for instance in data.get("database", {}).get("instances", [])
        if instance.get("DBClusterIdentifier") in {cluster["DBClusterIdentifier"] for cluster in clusters}
    ]
    separated = len(track_vpcs) == 1 and len(data_vpcs) == 1 and track_vpcs.isdisjoint(data_vpcs)
    attachments = [
        attachment
        for attachment in network.get("TransitGatewayAttachments", [])
        if attachment.get("ResourceType") == "vpc" and attachment.get("State") == "available"
    ]
    fabrics = {
        attachment["TransitGatewayId"]
        for attachment in attachments
        if attachment.get("ResourceId") in track_vpcs | data_vpcs
    }
    connected = separated and any(
        track_vpcs | data_vpcs
        <= {attachment["ResourceId"] for attachment in attachments if attachment["TransitGatewayId"] == fabric}
        for fabric in fabrics
    )
    distributions = data.get("edge", {}).get("distributions", [])
    hostname = urllib.parse.urlparse(evidence.get("endpoint", "")).hostname
    selected = [
        distribution
        for distribution in distributions
        if hostname == distribution.get("DomainName") or hostname in distribution.get("Aliases", {}).get("Items", [])
    ]
    distribution = selected[0] if len(selected) == 1 else (distributions[0] if len(distributions) == 1 else {})
    origins = distribution.get("Origins", {}).get("Items", [])
    console_buckets = {
        origin["DomainName"].split(".s3")[0] for origin in origins if ".s3" in origin.get("DomainName", "")
    }
    destinations = [
        destination.get("ExtendedS3DestinationDescription", destination.get("S3DestinationDescription", {}))
        for stream in data.get("stream", [])
        for destination in stream.get("Destinations", [])
    ]
    lake_buckets = {
        destination["BucketARN"].split(":::")[-1] for destination in destinations if destination.get("BucketARN")
    }
    functions = data.get("functions", [])
    finalizers = [
        function for function in functions if function.get("CodeSha256") == os.environ.get("FINALIZER_SHA256")
    ]
    telemetry_buckets = {
        function.get("Environment", {}).get("Variables", {}).get("TELEMETRY_BUCKET") for function in finalizers
    }
    telemetry_buckets.discard(None)
    buckets = data.get("buckets", {})
    bucket_errors = ["bucket:" + name for name in console_buckets | lake_buckets | telemetry_buckets]
    enabled = lambda name: [
        rule for rule in buckets.get(name, {}).get("lifecycle", []) if rule.get("Status") == "Enabled"
    ]
    core = bool(apps["race-api"] and apps["timing-engine"] and clusters and distribution)
    safe_instances = {"db.t3.medium", "db.t4g.medium", "db.r5.large", "db.r6g.large", "db.r7g.large", "db.r8g.large"}
    sized = (
        bool(clusters)
        and bool(instances)
        and all(
            cluster.get("ServerlessV2ScalingConfiguration")
            or all(
                instance.get("DBInstanceClass") in safe_instances
                for instance in instances
                if instance.get("DBClusterIdentifier") == cluster["DBClusterIdentifier"]
            )
            for cluster in clusters
        )
    )
    mark(
        "A1-01",
        console_buckets and all(enabled(name) for name in console_buckets),
        "Enabled lifecycle on every console origin bucket",
        "edge",
        *bucket_errors,
    )
    mark(
        "A1-02",
        lake_buckets and all(enabled(name) for name in lake_buckets),
        "Enabled lifecycle on every Firehose destination bucket",
        "stream",
        *bucket_errors,
    )
    mark("A1-03", sized, "Aurora Serverless v2 or medium/large provisioned instances", "database")
    mark(
        "A1-04",
        containers
        and all(
            0 < int(task.get("cpu", 0)) <= 1024 and 0 < int(task.get("memory", 0)) <= 4096 for task in application_tasks
        ),
        "Application tasks use at most 1 vCPU and 4 GiB",
        "containers",
    )
    mark(
        "A1-05",
        connected and len(network.get("TransitGateways", [])) == 1,
        "Single TGW with both application and database VPC attachments",
        "network",
        "containers",
        "database",
    )
    mark(
        "A1-06",
        destinations
        and all(
            destination.get("CompressionFormat", "UNCOMPRESSED") != "UNCOMPRESSED"
            or (
                1 <= destination.get("BufferingHints", {}).get("SizeInMBs", 0) <= 128
                and 0 <= destination.get("BufferingHints", {}).get("IntervalInSeconds", 999) <= 900
            )
            for destination in destinations
        ),
        "Firehose compression or bounded buffering",
        "stream",
    )
    mark("B1-01", separated, "ECS and Aurora occupy distinct VPCs", "network", "containers", "database")
    mark("B1-02", connected, "Both VPCs attach to the same available TGW", "network", "containers", "database")
    mark(
        "B1-03",
        clusters
        and any(cluster.get("DatabaseName") == "griddb" for cluster in clusters)
        and passed("tracks-spectator", "track-detail"),
        "Aurora griddb and six seeded tracks verified through the supplied API; driver count follows the original scoring proxy",
        "database",
    )
    fargate = all(
        task.get("runtimePlatform", {}).get("cpuArchitecture") == "ARM64"
        and task.get("runtimePlatform", {}).get("operatingSystemFamily", "LINUX") == "LINUX"
        and "FARGATE" in task.get("requiresCompatibilities", [])
        for task in application_tasks
    )
    mark(
        "B1-04",
        apps["race-api"] and apps["timing-engine"] and fargate,
        "Both applications use Linux ARM64 Fargate task definitions",
        "containers",
    )
    mark(
        "B1-05",
        any(
            {"director", "spectator"} <= {group["GroupName"] for group in pool["groups"]}
            for pool in data.get("identity", [])
        ),
        "Director and spectator groups in the same Cognito user pool",
        "identity",
    )
    behaviors = distribution.get("CacheBehaviors", {}).get("Items", []) + (
        [distribution["DefaultCacheBehavior"]] if distribution.get("DefaultCacheBehavior") else []
    )
    covered = bool(distribution)
    for path in ("/", "/api/tracks", "/health", "/transponder"):
        behavior = next(
            (item for item in behaviors if fnmatch.fnmatch(path.lstrip("/"), item.get("PathPattern", "*").lstrip("/"))),
            {},
        )
        origin = next((item for item in origins if item["Id"] == behavior.get("TargetOriginId")), {})
        covered = covered and bool(origin) and (path == "/" or ".s3" not in origin.get("DomainName", ""))
    mark("B1-06", covered, "CloudFront behaviors cover all four required paths", "edge")
    profiles = [profile for app in data.get("configuration", []) for profile in app["profiles"]]
    documents = [
        document for profile in profiles for document in profile.get("documents", []) if isinstance(document, dict)
    ]
    runtime_keys = {"DB_HOST", "DB_PORT", "DB_NAME", "EVENT_BUS_NAME", "LAP_STREAM_NAME", "SQS_QUEUE_URL"}
    rules = [document for document in documents if {"version", "min_lap_ms", "max_lap_ms"} <= document.keys()]
    config_ok = any(
        runtime_keys <= document.keys() and all(document[key] for key in runtime_keys) for document in documents
    ) and {"baseline-2026", "wet-track-2026"} <= {rule["version"] for rule in rules}
    config_ok = config_ok and any(
        deployment.get("State") == "COMPLETE"
        for app in data.get("configuration", [])
        for deployment in app["deployments"]
    )
    mark(
        "B1-07",
        config_ok,
        "Hosted runtime values, baseline and wet-track rules, and a completed deployment",
        "configuration",
    )
    last = (evidence.get("probe") or {}).get("last", {})
    current = (evidence.get("probe") or {}).get("current", {})
    progress = current if current.get("laps", 0) else last
    valid_laps = progress.get("laps", 0) >= 3 and progress.get("standings")
    transponders = [
        function
        for function in functions
        if any(function["FunctionArn"] in integration for integration in data.get("api", []))
    ]
    mark(
        "B1-08",
        transponders and valid_laps,
        "API Gateway POST route targets Lambda; accepted crossings visible in standings",
        "functions",
        "api",
    )
    queues = data.get("messaging", {}).get("queues", [])
    lap_queues = [queue for queue in queues if queue["url"].rsplit("/", 1)[-1] == "lap-events"]
    mark(
        "B1-09",
        lap_queues and apps["timing-engine"] and valid_laps,
        "lap-events queue, timing-engine service and populated standings",
        "messaging",
        "containers",
    )
    machines = data.get("workflow", {}).get("machines", [])
    machine_arns = {machine["stateMachineArn"] for machine in machines}
    start_rules = []
    for rule in data.get("workflow", {}).get("rules", []):
        pattern = json.loads(rule.get("EventPattern", "{}"))
        if (
            rule.get("State") == "ENABLED"
            and "gridline.race" in pattern.get("source", [])
            and "HeatStartRequested" in pattern.get("detail-type", [])
        ):
            start_rules.extend(target for target in rule.get("targets", []) if target.get("Arn") in machine_arns)
    mark("B1-10", start_rules, "Enabled gridline.race / HeatStartRequested rule targets Step Functions", "workflow")
    finalizer_arns = {function["FunctionArn"] for function in finalizers}
    finalizer_workflows = [
        machine for machine in machines if any(arn in machine.get("definition", "") for arn in finalizer_arns)
    ]
    mark(
        "B1-11",
        finalizer_workflows,
        "Workflow references a Lambda with the unchanged supplied finalizer ZIP hash",
        "workflow",
        "functions",
    )
    subscriptions = [
        subscription
        for subscription in data.get("messaging", {}).get("subscriptions", [])
        if subscription.get("Protocol") == "sqs"
        and any(queue["attributes"].get("QueueArn") == subscription.get("Endpoint") for queue in queues)
    ]
    mark(
        "B1-12",
        subscriptions and any("sns:publish" in machine.get("definition", "") for machine in machines),
        "Step Functions SNS publish and SNS-to-SQS subscription",
        "workflow",
        "messaging",
    )
    mark(
        "B1-13",
        destinations and any(destination.get("Prefix", "").startswith("laps/") for destination in destinations),
        "Firehose S3 destination uses laps/",
        "stream",
    )
    pipelines = data.get("pipeline", {}).get("pipelines", [])
    console_pipelines = [
        pipeline
        for pipeline in pipelines
        if any(
            action["actionTypeId"]["provider"] == "S3"
            and action.get("configuration", {}).get("S3ObjectKey", "").endswith("console.zip")
            for stage in pipeline["stages"]
            for action in stage["actions"]
        )
    ]
    build_projects = data.get("pipeline", {}).get("projects", {})
    deployed = any(
        any(
            "cloudfront"
            in build_projects.get(action.get("configuration", {}).get("ProjectName"), {})
            .get("source", {})
            .get("buildspec", "")
            and "create-invalidation"
            in build_projects.get(action.get("configuration", {}).get("ProjectName"), {})
            .get("source", {})
            .get("buildspec", "")
            for stage in pipeline["stages"]
            for action in stage["actions"]
            if action["actionTypeId"]["provider"] == "CodeBuild"
        )
        and pipeline.get("state", {}).get("stageStates")
        and all(
            stage.get("latestExecution", {}).get("status") == "Succeeded" for stage in pipeline["state"]["stageStates"]
        )
        for pipeline in console_pipelines
    )
    mark(
        "B1-14",
        deployed,
        "Console ZIP source, CloudFront invalidation in CodeBuild, and all pipeline stages succeeded",
        "pipeline",
    )
    telemetry = None
    if last.get("heat_id"):
        for bucket in telemetry_buckets:
            try:
                response = optional("s3", "get_object", Bucket=bucket, Key="telemetry/" + last["heat_id"] + ".json")
                if response.get("Body"):
                    candidate = json.loads(response["Body"].read(1024 * 1024))
                    if (
                        candidate.get("heat_id") == last["heat_id"]
                        and candidate.get("track_id") == "track-01"
                        and candidate.get("started_at")
                        and candidate.get("finished_at")
                        and candidate.get("rules_version")
                        and isinstance(candidate.get("label"), str)
                        and isinstance(candidate.get("lap_count"), int)
                        and candidate["lap_count"] >= 0
                        and isinstance(candidate.get("podium"), list)
                    ):
                        telemetry = candidate
            except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
                errors["telemetry"] = str(error)[:240]
    mark("B1-15", telemetry, "Telemetry object for the completed assessment heat", "functions", "telemetry")
    mark(
        "C1-01",
        passed("tracks-unauth", "heats-spectator-forbidden"),
        "Current unauthenticated and spectator authorization checks",
    )
    mark("C1-02", passed("tracks-unauth"), "Current unauthenticated tracks request returns 401")
    mark("C1-03", passed("heats-spectator-forbidden"), "Current spectator heat creation returns 403")
    secret_refs = all(
        {"DB_USERNAME", "DB_PASSWORD"}
        <= {
            secret["name"]
            for secret in container.get("secrets", [])
            if ":secretsmanager:" in secret.get("valueFrom", "")
        }
        and not any(
            variable.get("name") == "DB_PASSWORD" and variable.get("value")
            for variable in container.get("environment", [])
        )
        for container in containers
    )
    plain_lambda_passwords = any(
        function.get("Environment", {}).get("Variables", {}).get("DB_PASSWORD")
        for function in functions
        if function not in finalizers
    )
    mark(
        "C1-04",
        containers and secret_refs and clusters and not plain_lambda_passwords,
        "ECS database credentials use Secrets Manager references; finalizer environment exception follows the supplied brief",
        "containers",
        "database",
        "functions",
    )
    private_timing = bool(apps["timing-engine"]) and all(
        not service.get("loadBalancers")
        and service.get("networkConfiguration", {}).get("awsvpcConfiguration", {}).get("assignPublicIp") == "DISABLED"
        for service, _ in apps["timing-engine"]
    )
    mark("C1-05", private_timing, "Timing-engine has no public address or load balancer attachment", "containers")
    private_console = bool(console_buckets) and all(
        all(
            buckets.get(name, {}).get("public", {}).get(key)
            for key in ("BlockPublicAcls", "IgnorePublicAcls", "BlockPublicPolicy", "RestrictPublicBuckets")
        )
        for name in console_buckets
    )
    mark("C1-06", private_console, "Every console origin bucket blocks public access", "edge", *bucket_errors)
    mark(
        "C1-07",
        separated and all(not instance.get("PubliclyAccessible") for instance in instances),
        "Aurora subnet group is in the separate data VPC with private instances",
        "database",
        "network",
        "containers",
    )
    private_links = [
        endpoint
        for endpoint in network.get("VpcEndpoints", [])
        if endpoint.get("VpcEndpointType") in ("Interface", "GatewayLoadBalancer")
    ]
    peering = [
        connection
        for connection in network.get("VpcPeeringConnections", [])
        if connection.get("Status", {}).get("Code") == "active"
        and {connection.get("AccepterVpcInfo", {}).get("VpcId"), connection.get("RequesterVpcInfo", {}).get("VpcId")}
        == track_vpcs | data_vpcs
    ]
    mark(
        "C1-08",
        connected and not peering and not private_links,
        "TGW database path without VPC peering or PrivateLink",
        "network",
        "database",
        "containers",
    )
    banned = data.get("banned", {})
    mark(
        "C1-09",
        core and not private_links and not any(banned.values()),
        "No Route 53, PrivateLink, ElastiCache or VPC Lattice resources",
        "banned",
        "network",
    )
    required_roles = {task.get("taskRoleArn") for task in application_tasks}
    required_roles.update(function["Role"] for function in functions)
    required_roles.discard(None)
    broad = []
    wildcard_actions = {
        "ecr:GetAuthorizationToken",
        "logs:CreateLogGroup",
        "ec2:CreateNetworkInterface",
        "ec2:DeleteNetworkInterface",
    }
    for arn in required_roles:
        for document in data.get("policies", {}).get(arn, []):
            for statement in document.get("Statement", []):
                if statement.get("Effect") != "Allow":
                    continue
                actions = statement.get("Action", [])
                actions = [actions] if isinstance(actions, str) else actions
                resources = statement.get("Resource", [])
                resources = [resources] if isinstance(resources, str) else resources
                if (
                    statement.get("NotAction")
                    or statement.get("NotResource")
                    or any("*" in action for action in actions)
                    or (
                        "*" in resources
                        and any(
                            action not in wildcard_actions and not action.startswith("ec2:Describe")
                            for action in actions
                        )
                    )
                ):
                    broad.append(arn.rsplit("/", 1)[-1])
    mark(
        "C1-10",
        required_roles and all(data.get("policies", {}).get(arn) for arn in required_roles) and not broad,
        "Broad application roles: " + ", ".join(sorted(set(broad)))
        if broad
        else "Application role actions and resources are scoped",
        "containers",
        "functions",
        *("role:" + arn for arn in required_roles),
    )
    ssh = [
        group["GroupId"]
        for group in network.get("SecurityGroups", [])
        for rule in group.get("IpPermissions", [])
        if (
            rule.get("IpProtocol") == "-1"
            or (rule.get("IpProtocol") in ("tcp", "6") and rule.get("FromPort", 0) <= 22 <= rule.get("ToPort", 65535))
        )
        and any(block.get("CidrIp") == "0.0.0.0/0" for block in rule.get("IpRanges", []))
    ]
    mark(
        "C1-11",
        network.get("SecurityGroups") and not ssh,
        "World-open SSH security groups: " + ", ".join(ssh) if ssh else "No security group exposes SSH to 0.0.0.0/0",
        "network",
    )
    regional_arns = (
        [cluster.get("DBClusterArn", "") for cluster in clusters]
        + [service["serviceArn"] for service in application_services]
        + [function["FunctionArn"] for function in functions]
    )
    region_ok = (
        core
        and all(arn.split(":")[3] == REGION for arn in regional_arns if arn)
        and all(
            buckets.get(name, {}).get("region") == REGION for name in console_buckets | lake_buckets | telemetry_buckets
        )
    )
    mark(
        "C1-12",
        region_ok,
        "Marked regional resources and application buckets use eu-central-1",
        "database",
        "containers",
        "functions",
        *bucket_errors,
    )
    rule_profiles = [
        profile
        for profile in profiles
        if any(isinstance(document, dict) and "min_lap_ms" in document for document in profile.get("documents", []))
    ]
    mark(
        "C1-13",
        rule_profiles and all(profile.get("Validators") for profile in rule_profiles),
        "All timing-rule profiles have AppConfig validators",
        "configuration",
    )
    mark(
        "C1-14",
        clusters and all(cluster.get("StorageEncrypted") for cluster in clusters),
        "Aurora storage encryption enabled",
        "database",
    )
    multi_az = bool(clusters) and all(
        len(
            {
                instance.get("AvailabilityZone")
                for instance in instances
                if instance.get("DBClusterIdentifier") == cluster["DBClusterIdentifier"]
            }
        )
        >= 2
        for cluster in clusters
    )
    mark("D1-01", multi_az, "Aurora instances occupy at least two availability zones", "database")
    service_azs = all(
        service.get("desiredCount", 0) >= 2
        or len(
            {
                subnets[subnet]["AvailabilityZone"]
                for subnet in service.get("networkConfiguration", {}).get("awsvpcConfiguration", {}).get("subnets", [])
                if subnet in subnets
            }
        )
        >= 2
        for service in application_services
    )
    mark(
        "D1-02",
        apps["race-api"] and apps["timing-engine"] and service_azs,
        "Both ECS services span AZs or maintain at least two tasks",
        "containers",
        "network",
    )
    mark(
        "D1-03",
        last.get("started") and last.get("running") and last.get("finished") and finalizer_workflows,
        "Assessment heat observed RUNNING then FINISHED through the configured workflow",
        "workflow",
    )
    mark("D1-04", valid_laps, "Accepted transponder laps became visible in standings")
    all_checks = len(evidence.get("checks", {})) == 13 and all(check.get("ok") for check in evidence["checks"].values())
    mark("D1-05", all_checks, "All 13 current functional checks pass")
    log_names = {group["logGroupName"] for group in data.get("logs", [])}
    ecs_logging = containers and all(
        container.get("logConfiguration", {}).get("logDriver") == "awslogs"
        and container["logConfiguration"].get("options", {}).get("awslogs-group") in log_names
        for container in containers
    )
    lambda_logging = functions and all(
        function.get("LoggingConfig", {}).get("LogGroup", "/aws/lambda/" + function["FunctionName"]) in log_names
        for function in functions
    )
    mark(
        "D1-06",
        ecs_logging and lambda_logging,
        "ECS awslogs and application Lambda log groups exist",
        "logs",
        "containers",
        "functions",
    )
    mark(
        "D1-07",
        lap_queues and all(0 < int(queue["attributes"].get("VisibilityTimeout", 0)) <= 300 for queue in lap_queues),
        "lap-events visibility timeout is bounded at 1–300 seconds",
        "messaging",
    )
    success_rate = sum(check.get("ok", False) for check in evidence.get("checks", {}).values()) / 13
    for identifier, threshold in (("D1-08", 0.9), ("D1-09", 0.8), ("D1-10", 0.7)):
        mark(
            identifier,
            len(evidence.get("checks", {})) == 13 and success_rate >= threshold,
            "Current functional success rate: " + str(round(success_rate * 100, 1)) + "%",
        )
    https = (
        len(distributions) == 1
        and distribution.get("Enabled")
        and all(behavior.get("ViewerProtocolPolicy") in ("https-only", "redirect-to-https") for behavior in behaviors)
    )
    mark("E1-01", https, "One enabled CloudFront distribution enforces viewer HTTPS", "edge")
    latencies = evidence.get("latencies", {})

    def p95(identifier, limit):
        samples = sorted(latencies.get(identifier, []))
        return len(samples) >= 3 and samples[math.ceil(len(samples) * 0.95) - 1] <= limit

    mark(
        "E1-02",
        passed("leaderboard") and p95("leaderboard", 300),
        "Leaderboard p95 over the latest 20 attempts is at most 300 ms (minimum three samples)",
    )
    other_ids = [identifier for identifier in evidence.get("checks", {}) if identifier != "leaderboard"]
    mark(
        "E1-03",
        len(other_ids) == 12 and all(p95(identifier, 500) for identifier in other_ids),
        "Other API p95 values over the latest 20 attempts are at most 500 ms (minimum three samples)",
    )
    service_ids = {
        "service/" + service["clusterArn"].split("/")[-1] + "/" + service["serviceName"]
        for service in application_services
    }
    targets = [
        target
        for target in data.get("scaling", {}).get("targets", [])
        if target["ResourceId"] in service_ids
        and target["ScalableDimension"] == "ecs:service:DesiredCount"
        and target["MaxCapacity"] > target["MinCapacity"]
    ]
    policies = [
        item
        for item in data.get("scaling", {}).get("policies", [])
        if item["ResourceId"] in {target["ResourceId"] for target in targets}
    ]
    mark(
        "E1-04",
        targets and policies,
        "Application ECS service has a scalable target and scaling policy",
        "scaling",
        "containers",
    )
    default = distribution.get("DefaultCacheBehavior", {})
    default_origin = next((origin for origin in origins if origin["Id"] == default.get("TargetOriginId")), {})
    cache_policy = data.get("edge", {}).get("policies", {}).get(default.get("CachePolicyId"), {})
    mark(
        "E1-05",
        ".s3" in default_origin.get("DomainName", "")
        and (cache_policy.get("DefaultTTL", 0) > 0 or default.get("DefaultTTL", 0) > 0),
        "Default console behavior targets S3 with a positive cache TTL",
        "edge",
    )
    mark(
        "E1-06",
        progress.get("min_rejected") and progress.get("max_rejected"),
        "Both too-short and too-long lap timestamps returned 422",
    )
    transitions = any(
        rule.get("Transitions") or rule.get("NoncurrentVersionTransitions")
        for name in lake_buckets | telemetry_buckets
        for rule in enabled(name)
    )
    mark("F1-01", transitions, "Enabled IA/archive lifecycle transitions on a laps or telemetry bucket", *bucket_errors)
    spot = any(
        strategy.get("capacityProvider") == "FARGATE_SPOT"
        for service in application_services
        for strategy in service.get("capacityProviderStrategy", [])
    ) or any(
        "FARGATE_SPOT" in cluster.get("capacityProviders", [])
        for cluster in data.get("containers", {}).get("clusters", [])
    )
    mark("F1-02", spot, "Fargate Spot is configured on an application service or cluster", "containers")
    scale_in = any(
        (
            item.get("PolicyType") == "TargetTrackingScaling"
            and not item.get("TargetTrackingScalingPolicyConfiguration", {}).get("DisableScaleIn", False)
        )
        or any(
            step.get("ScalingAdjustment", 0) < 0
            for step in item.get("StepScalingPolicyConfiguration", {}).get("StepAdjustments", [])
        )
        for item in policies
    )
    mark("F1-03", targets and scale_in, "Application autoscaling permits scale-in", "scaling", "containers")
    footprint = (
        len(services)
        + len(data.get("containers", {}).get("clusters", []))
        + len(clusters)
        + len(distributions)
        + len(functions)
        + len(machines)
    )
    mark(
        "F1-04",
        core
        and len(apps["race-api"]) == 1
        and len(apps["timing-engine"]) == 1
        and len(clusters) == 1
        and len(distributions) == 1
        and footprint <= 16,
        "Core always-on footprint: " + str(footprint) + " resources; assessor excluded",
        "containers",
        "database",
        "edge",
        "functions",
        "workflow",
    )
    mark("G1-01", all_checks, "All 13 current scoring checks pass during marking")
    notification = False
    if evidence.get("active"):
        for subscription in subscriptions:
            queue = next(queue for queue in queues if queue["attributes"].get("QueueArn") == subscription["Endpoint"])
            try:
                for message in fetch(
                    "sqs",
                    "receive_message",
                    "Messages",
                    QueueUrl=queue["url"],
                    MaxNumberOfMessages=10,
                    VisibilityTimeout=0,
                    WaitTimeSeconds=0,
                ):
                    body = json.loads(message["Body"])
                    if "Message" in body:
                        body = json.loads(body["Message"])
                    notification |= (
                        body.get("event") == "heat.finished"
                        and body.get("status") == "FINISHED"
                        and body.get("track_id") == "track-01"
                        and isinstance(body.get("lap_count"), int)
                        and body["lap_count"] >= 0
                        and isinstance(body.get("label"), str)
                        and bool(body.get("heat_id"))
                        and bool(body.get("ts"))
                    )
            except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
                errors["notification"] = str(error)[:240]
    mark(
        "G1-02",
        notification,
        "Valid heat.finished notification observed in a subscribed SQS queue without deleting it",
        "messaging",
        "notification",
    )
    mark(
        "G1-03",
        last.get("started")
        and last.get("running")
        and last.get("finished")
        and last.get("laps", 0) >= 6
        and last.get("standings")
        and telemetry
        and telemetry.get("lap_count", 0) >= 6
        and len(telemetry.get("podium", [])) >= 3,
        "Assessment heat completed with six accepted laps, standings and persisted telemetry",
        "telemetry",
    )
    url_ok = False
    if evidence.get("active") and evidence.get("endpoint"):
        try:
            root_code, _ = request(evidence["endpoint"], "GET", "/", group="")
            url_ok = (
                len(selected) == 1 and root_code == 200 and passed("health", "tracks-unauth", "transponder-not-running")
            )
        except (OSError, ValueError) as error:
            errors["endpoint"] = str(error)[:240]
    mark("G1-04", url_ok, "Submitted endpoint serves console, health, protected API and transponder paths", "endpoint")
    return results


def handler(event, context):
    authorization = event.get("headers", {}).get("authorization", "")
    if not hmac.compare_digest(authorization, "Bearer " + os.environ["ASSESSOR_TOKEN"]):
        return {"statusCode": 403, "body": "Forbidden"}
    try:
        body = event.get("body", "{}")
        if event.get("isBase64Encoded"):
            body = base64.b64decode(body)
        evidence = json.loads(body)
        evidence["probe"] = advance(evidence)
        report = {"results": assess(snapshot(), evidence), "probe": evidence["probe"]}
        return {"statusCode": 200, "headers": {"content-type": "application/json"}, "body": json.dumps(report)}
    except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
        return {"statusCode": 500, "body": json.dumps({"error": str(error)[:240]})}
