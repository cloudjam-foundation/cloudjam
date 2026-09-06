import base64
import concurrent.futures
import fnmatch
import hmac
import json
import os
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request

import boto3
from botocore.config import Config
from botocore.exceptions import BotoCoreError, ClientError

REGION = os.environ.get("AWS_REGION", "eu-central-1")
PREFIX = "cloudjam-vitalforge-assessor-"
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
            "ParameterNotFound",
        ):
            return {}
        raise


def kubernetes(cluster):
    name = cluster["name"]
    role = os.environ["ASSESSOR_ROLE"]
    try:
        fetch("eks", "create_access_entry", clusterName=name, principalArn=role, type="STANDARD")
    except ClientError as error:
        if error.response["Error"]["Code"] != "ResourceInUseException":
            raise
    fetch(
        "eks",
        "associate_access_policy",
        clusterName=name,
        principalArn=role,
        policyArn="arn:aws:eks::aws:cluster-access-policy/AmazonEKSAdminViewPolicy",
        accessScope={"type": "cluster"},
    )
    sts = client("sts")
    request = sts._convert_to_request_dict(
        {}, sts.meta.service_model.operation_model("GetCallerIdentity"), sts.meta.endpoint_url
    )
    request["method"] = "GET"
    request["headers"]["x-k8s-aws-id"] = name
    url = sts._request_signer.generate_presigned_url(
        request, region_name=REGION, expires_in=60, operation_name="GetCallerIdentity"
    )
    token = "k8s-aws-v1." + base64.urlsafe_b64encode(url.encode()).decode().rstrip("=")
    context = ssl.create_default_context(cadata=base64.b64decode(cluster["certificateAuthority"]["data"]).decode())
    data = {}
    for key, path in {
        "deployments": "/apis/apps/v1/deployments",
        "pods": "/api/v1/pods",
        "nodes": "/api/v1/nodes",
        "accounts": "/api/v1/serviceaccounts",
        "hpa": "/apis/autoscaling/v2/horizontalpodautoscalers",
        "volumes": "/api/v1/persistentvolumes",
    }.items():
        req = urllib.request.Request(cluster["endpoint"] + path, headers={"Authorization": "Bearer " + token})
        with urllib.request.urlopen(req, context=context, timeout=4) as response:
            data[key] = json.load(response).get("items", [])
    return data


def inventory(section):
    if section == "network":
        return {
            key: fetch("ec2", operation, key)
            for key, operation in (
                ("SecurityGroups", "describe_security_groups"),
                ("VpcPeeringConnections", "describe_vpc_peering_connections"),
                ("TransitGatewayAttachments", "describe_transit_gateway_attachments"),
                ("RouteTables", "describe_route_tables"),
            )
        }
    if section == "containers":
        clusters = []
        selected = (
            optional("ssm", "get_parameter", Name="/cloudjam/vitalforge/day2/cluster-name")
            .get("Parameter", {})
            .get("Value")
        )
        for name in fetch("eks", "list_clusters", "clusters"):
            if name != selected and "vitalforge" not in name.lower():
                continue
            cluster = fetch("eks", "describe_cluster", name=name)["cluster"]
            cluster["nodegroups"] = [
                fetch("eks", "describe_nodegroup", clusterName=name, nodegroupName=group)["nodegroup"]
                for group in fetch("eks", "list_nodegroups", "nodegroups", clusterName=name)
            ]
            cluster["podRoles"] = [
                fetch("eks", "describe_pod_identity_association", clusterName=name, associationId=a["associationId"])[
                    "association"
                ]
                for a in fetch("eks", "list_pod_identity_associations", "associations", clusterName=name)
            ]
            try:
                cluster["kubernetes"] = kubernetes(cluster)
            except (BotoCoreError, ClientError, OSError, ValueError) as error:
                cluster["error"] = str(error)[:240]
            clusters.append(cluster)
        return clusters
    if section == "database":
        tables = {}
        for name in fetch("dynamodb", "list_tables", "TableNames"):
            if name.startswith("vitalforge-"):
                table = fetch("dynamodb", "describe_table", TableName=name)["Table"]
                if name.endswith("-users"):
                    table["users"] = fetch("dynamodb", "scan", TableName=name, ConsistentRead=True).get("Items", [])
                tables[name] = table
        return {
            "tables": tables,
            "rds": fetch("rds", "describe_db_instances", "DBInstances"),
            "aurora": fetch("rds", "describe_db_clusters", "DBClusters"),
        }
    if section == "files":
        systems = fetch("efs", "describe_file_systems", "FileSystems")
        for system in systems:
            system["targets"] = fetch(
                "efs", "describe_mount_targets", "MountTargets", FileSystemId=system["FileSystemId"]
            )
            system["points"] = fetch(
                "efs", "describe_access_points", "AccessPoints", FileSystemId=system["FileSystemId"]
            )
            for target in system["targets"]:
                target["groups"] = fetch(
                    "efs",
                    "describe_mount_target_security_groups",
                    "SecurityGroups",
                    MountTargetId=target["MountTargetId"],
                )
        return systems
    if section == "functions":
        functions = [
            f
            for f in fetch("lambda", "list_functions", "Functions")
            if not f["FunctionName"].startswith(PREFIX)
            and (
                "vitalforge" in f["FunctionName"].lower()
                or f.get("CodeSha256") in (os.environ["SLEEP_SHA256"], os.environ["EXPORT_SHA256"])
            )
        ]
        for function in functions:
            function["accountConcurrency"] = fetch("lambda", "get_account_settings")["AccountLimit"][
                "ConcurrentExecutions"
            ]
            function["concurrency"] = fetch(
                "lambda", "get_function_concurrency", FunctionName=function["FunctionName"]
            ).get("ReservedConcurrentExecutions")
        return functions
    if section == "edge":
        distributions = fetch("cloudfront", "list_distributions").get("DistributionList", {}).get("Items", [])
        policies = {}
        for distribution in distributions:
            policy = distribution.get("DefaultCacheBehavior", {}).get("CachePolicyId")
            if policy:
                policies[policy] = fetch("cloudfront", "get_cache_policy", Id=policy)["CachePolicy"][
                    "CachePolicyConfig"
                ]
        return {"distributions": distributions, "policies": policies}
    if section == "api":
        apis = fetch("apigatewayv2", "get_apis", "Items")
        for api in apis:
            api["routes"] = fetch("apigatewayv2", "get_routes", "Items", ApiId=api["ApiId"])
            api["integrations"] = fetch("apigatewayv2", "get_integrations", "Items", ApiId=api["ApiId"])
        balancers = fetch("elbv2", "describe_load_balancers", "LoadBalancers")
        groups = fetch("elbv2", "describe_target_groups", "TargetGroups")
        for group in groups:
            group["targets"] = fetch(
                "elbv2", "describe_target_health", "TargetHealthDescriptions", TargetGroupArn=group["TargetGroupArn"]
            )
        return {"apis": apis, "balancers": balancers, "groups": groups}
    if section == "identity":
        return [
            fetch("cognito-idp", "list_groups", "Groups", UserPoolId=pool["Id"])
            for pool in fetch("cognito-idp", "list_user_pools", "UserPools", MaxResults=60)
        ]
    if section == "configuration":
        return {
            p["Name"]: p["Value"]
            for p in fetch("ssm", "get_parameters_by_path", "Parameters", Path="/vitalforge/", Recursive=True)
        }
    if section == "lattice":
        services = fetch("vpc-lattice", "list_services", "items")
        networks = fetch("vpc-lattice", "list_service_networks", "items")
        for network in networks:
            network["vpcs"] = fetch(
                "vpc-lattice", "list_service_network_vpc_associations", "items", serviceNetworkIdentifier=network["id"]
            )
            network["services"] = fetch(
                "vpc-lattice",
                "list_service_network_service_associations",
                "items",
                serviceNetworkIdentifier=network["id"],
            )
        groups = fetch("vpc-lattice", "list_target_groups", "items")
        for group in groups:
            group["targets"] = fetch("vpc-lattice", "list_targets", "items", targetGroupIdentifier=group["id"])
        zones = [
            z for z in fetch("route53", "list_hosted_zones", "HostedZones") if z.get("Config", {}).get("PrivateZone")
        ]
        for zone in zones:
            zone["records"] = fetch(
                "route53", "list_resource_record_sets", "ResourceRecordSets", HostedZoneId=zone["Id"]
            )
            zone["vpcs"] = fetch("route53", "get_hosted_zone", Id=zone["Id"]).get("VPCs", [])
        return {"services": services, "networks": networks, "groups": groups, "zones": zones}
    if section == "workflow":
        schedules = [
            fetch("scheduler", "get_schedule", Name=s["Name"], GroupName=s["GroupName"])
            for s in fetch("scheduler", "list_schedules", "Schedules")
        ]
        rules = fetch("events", "list_rules", "Rules")
        for rule in rules:
            rule["targets"] = fetch("events", "list_targets_by_rule", "Targets", Rule=rule["Name"])
        return {"schedules": schedules, "rules": rules}
    if section == "audit":
        trails = fetch("cloudtrail", "describe_trails", "trailList", includeShadowTrails=False)
        for trail in trails:
            trail["status"] = fetch("cloudtrail", "get_trail_status", Name=trail["TrailARN"])
            trail["selectors"] = fetch("cloudtrail", "get_event_selectors", TrailName=trail["TrailARN"])
        return trails
    if section == "pipeline":
        pipelines = []
        for summary in fetch("codepipeline", "list_pipelines", "pipelines"):
            pipeline = fetch("codepipeline", "get_pipeline", name=summary["name"])["pipeline"]
            pipeline["state"] = fetch("codepipeline", "get_pipeline_state", name=summary["name"])
            names = [
                a["configuration"]["ProjectName"]
                for stage in pipeline["stages"]
                for a in stage["actions"]
                if a["actionTypeId"]["provider"] == "CodeBuild"
            ]
            pipeline["projects"] = fetch("codebuild", "batch_get_projects", "projects", names=names) if names else []
            pipelines.append(pipeline)
        return pipelines
    if section == "logs":
        return fetch("logs", "describe_log_groups", "logGroups")
    raise ValueError(section)


def snapshot():
    global CACHE, LOADED
    if CACHE is not None and time.time() - LOADED < 120:
        return CACHE
    for service in (
        "ec2",
        "eks",
        "sts",
        "dynamodb",
        "rds",
        "efs",
        "lambda",
        "cloudfront",
        "apigatewayv2",
        "elbv2",
        "cognito-idp",
        "ssm",
        "vpc-lattice",
        "route53",
        "scheduler",
        "events",
        "cloudtrail",
        "codepipeline",
        "codebuild",
        "logs",
        "s3",
        "iam",
    ):
        client(service)
    data, errors = {}, {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=12) as executor:
        futures = {
            executor.submit(inventory, section): section
            for section in (
                "network",
                "containers",
                "database",
                "files",
                "functions",
                "edge",
                "api",
                "identity",
                "configuration",
                "lattice",
                "workflow",
                "audit",
                "pipeline",
                "logs",
            )
        }
        for future in concurrent.futures.as_completed(futures):
            try:
                data[futures[future]] = future.result()
            except (BotoCoreError, ClientError, ValueError, KeyError, TypeError, OSError) as error:
                errors[futures[future]] = str(error)[:240]
    data["errors"], data["buckets"], data["policies"] = errors, {}, {}
    buckets = {
        o["DomainName"].split(".s3")[0]
        for d in data.get("edge", {}).get("distributions", [])
        for o in d.get("Origins", {}).get("Items", [])
        if ".s3" in o["DomainName"]
    }
    buckets.update(t["S3BucketName"] for t in data.get("audit", []))
    for name in buckets:
        try:
            data["buckets"][name] = {
                "lifecycle": optional("s3", "get_bucket_lifecycle_configuration", Bucket=name).get("Rules", []),
                "public": optional("s3", "get_public_access_block", Bucket=name).get(
                    "PublicAccessBlockConfiguration", {}
                ),
                "region": fetch("s3", "get_bucket_location", Bucket=name).get("LocationConstraint") or "us-east-1",
            }
        except (BotoCoreError, ClientError, OSError) as error:
            errors["bucket:" + name] = str(error)[:240]
    roles = {f["Role"] for f in data.get("functions", [])}
    for cluster in data.get("containers", []):
        roles.update(a["roleArn"] for a in cluster.get("podRoles", []))
        roles.update(
            a.get("metadata", {}).get("annotations", {}).get("eks.amazonaws.com/role-arn")
            for a in cluster.get("kubernetes", {}).get("accounts", [])
        )
    for arn in roles - {None}:
        try:
            name = arn.split("/")[-1]
            policies = [
                fetch("iam", "get_role_policy", RoleName=name, PolicyName=p)["PolicyDocument"]
                for p in fetch("iam", "list_role_policies", "PolicyNames", RoleName=name)
            ]
            for attached in fetch("iam", "list_attached_role_policies", "AttachedPolicies", RoleName=name):
                policy = fetch("iam", "get_policy", PolicyArn=attached["PolicyArn"])["Policy"]
                policies.append(
                    fetch(
                        "iam",
                        "get_policy_version",
                        PolicyArn=attached["PolicyArn"],
                        VersionId=policy["DefaultVersionId"],
                    )["PolicyVersion"]["Document"]
                )
            data["policies"][arn] = policies
        except (BotoCoreError, ClientError, OSError) as error:
            errors["role:" + arn] = str(error)[:240]
    CACHE, LOADED = data, time.time()
    return data


def request(base, method, path, body=None, group="coach"):
    headers = {"Accept": "application/json", "X-CloudJam-Check": "architecture"}
    if group:
        headers.update({"X-Dev-Sub": "scoring-coach" if group == "coach" else "member-demo-1", "X-Dev-Groups": group})
    payload = json.dumps(body).encode() if body is not None else None
    if payload is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base.rstrip("/") + path, data=payload, method=method, headers=headers)
    try:
        response = urllib.request.urlopen(req, timeout=3)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        raw = response.read(1048576)
        try:
            data = json.loads(raw)
        except ValueError:
            data = None
        return response.status, data


def advance(evidence):
    probe = evidence.get("probe") or {}
    if not evidence.get("active") or not evidence.get("endpoint"):
        return probe
    base = evidence["endpoint"]
    try:
        status, dashboard = request(base, "GET", "/api/members/member-demo-1/dashboard")
        probe["dashboard"] = dashboard if status == 200 else None
        status, export = request(base, "GET", "/api/export/member-demo-1/latest")
        previous = probe.get("export") or {}
        probe["overwritten"] = bool(
            probe.get("overwritten")
            or status == 200
            and previous.get("generated_at")
            and export.get("generated_at") != previous["generated_at"]
        )
        probe["export"] = export if status == 200 else None
        version = (dashboard or {}).get("analytics", {}).get("thresholds_version")
        versions = probe.setdefault("versions", [])
        if version and version not in versions:
            versions.append(version)
        probe["sleep_missing"] = request(base, "POST", "/sleep", {}, "")[0] == 400
        probe["sleep_unknown"] = (
            request(
                base,
                "POST",
                "/sleep",
                {
                    "user_id": "cloudjam-unknown-member",
                    "started_at": "2026-09-01T00:00:00Z",
                    "ended_at": "2026-09-01T08:00:00Z",
                },
                "",
            )[0]
            == 422
        )
        probe["console"] = request(base, "GET", "/", group="")[0] == 200
        probe.pop("error", None)
    except (OSError, ValueError, TypeError, KeyError) as error:
        probe["error"] = str(error)[:240]
    return probe


def assess(data, evidence):
    results, errors = {}, data.get("errors", {})

    def mark(identifier, passed, detail, *sections):
        unavailable = [section + ": " + errors[section] for section in sections if section in errors]
        results[identifier] = {
            "ok": bool(passed) and not unavailable,
            "detail": "; ".join(unavailable) if unavailable else detail,
        }

    def passed(*ids):
        return all(evidence.get("checks", {}).get(identifier, {}).get("ok", False) for identifier in ids)

    network = data.get("network", {})
    groups = {s["GroupId"]: s for s in network.get("SecurityGroups", [])}
    clusters = []
    apps = {name: [] for name in ("wellness-api", "analytics-engine")}
    pods, nodes, volumes, hpas, accounts, pod_roles = [], [], [], [], [], []
    for cluster in data.get("containers", []):
        k8s = cluster.get("kubernetes", {})
        found = False
        for deployment in k8s.get("deployments", []):
            for name, deployments in apps.items():
                if any(
                    name in (c.get("name", "") + " " + c.get("image", ""))
                    for c in deployment["spec"]["template"]["spec"].get("containers", [])
                ):
                    deployments.append(deployment)
                    found = True
        if found:
            clusters.append(cluster)
            pods.extend(k8s.get("pods", []))
            nodes.extend(k8s.get("nodes", []))
            volumes.extend(k8s.get("volumes", []))
            hpas.extend(k8s.get("hpa", []))
            accounts.extend(k8s.get("accounts", []))
            pod_roles.extend(cluster.get("podRoles", []))
    workloads = [d for items in apps.values() for d in items]
    app_names = {d["metadata"]["name"] for d in workloads}
    app_vpcs = {c["resourcesVpcConfig"]["vpcId"] for c in clusters}
    nodegroups = [g for c in clusters for g in c.get("nodegroups", [])]
    app_pods = [
        p
        for p in pods
        if any(
            name in (c.get("name", "") + " " + c.get("image", ""))
            for c in p["spec"].get("containers", [])
            for name in apps
        )
    ]
    analytics_pods = [
        p
        for p in app_pods
        if any(
            "analytics-engine" in (c.get("name", "") + " " + c.get("image", ""))
            for c in p["spec"].get("containers", [])
        )
    ]
    analytics_ips = {p.get("status", {}).get("podIP") for p in analytics_pods} - {None}
    node_map = {n["metadata"]["name"]: n for n in nodes}
    arm = all(
        node_map.get(p["spec"].get("nodeName"), {}).get("metadata", {}).get("labels", {}).get("kubernetes.io/arch")
        == "arm64"
        for p in app_pods
    )
    running = bool(
        all(apps.values())
        and app_pods
        and arm
        and all(d.get("status", {}).get("availableReplicas", 0) >= 1 for d in workloads)
    )
    container_detail = f"applications={sorted(app_names)}, ARM64 pods={len(app_pods) if arm else 0}; " + "; ".join(
        c.get("error", "") for c in data.get("containers", []) if c.get("error")
    )
    tables = data.get("database", {}).get("tables", {})
    expected = {"vitalforge-" + n for n in ("users", "meals", "workouts", "sleep")}
    on_demand = expected <= tables.keys() and all(
        tables[n].get("BillingModeSummary", {}).get("BillingMode") == "PAY_PER_REQUEST" for n in expected
    )
    mark(
        "A1-01",
        on_demand,
        f"On-demand application tables: {sum(t.get('BillingModeSummary', {}).get('BillingMode') == 'PAY_PER_REQUEST' for t in tables.values())}/4",
        "database",
    )
    sized = bool(nodegroups) and all(
        g.get("instanceTypes") and all(t.endswith((".medium", ".large", ".xlarge")) for t in g["instanceTypes"])
        for g in nodegroups
    )
    mark(
        "A1-02",
        running and sized,
        container_detail + f"; node types={[g.get('instanceTypes') for g in nodegroups]}",
        "containers",
    )
    host = urllib.parse.urlparse(evidence.get("endpoint", "")).hostname
    distributions = [d for d in data.get("edge", {}).get("distributions", []) if d.get("DomainName") == host]
    distribution = distributions[0] if len(distributions) == 1 else {}
    edge = bool(
        distribution.get("Enabled")
        and host
        and host.endswith(".cloudfront.net")
        and evidence.get("endpoint", "").startswith("https://")
    )
    mark("A1-03", edge, f"Submitted CloudFront distribution: {host}", "edge")
    functions = data.get("functions", [])
    sleepers = [f for f in functions if f.get("CodeSha256") == os.environ["SLEEP_SHA256"]]
    exporters = [f for f in functions if f.get("CodeSha256") == os.environ["EXPORT_SHA256"]]
    point_arns = {p["Arn"] for f in exporters for p in f.get("FileSystemConfigs", [])}
    systems = [
        f
        for f in data.get("files", [])
        if any(p["AccessPointArn"] in point_arns for p in f["points"])
        or any(f["FileSystemId"] in v.get("spec", {}).get("csi", {}).get("volumeHandle", "") for v in volumes)
    ]
    targets = [t for f in systems for t in f["targets"]]
    files_vpcs = {t["VpcId"] for t in targets}
    peers = [
        p
        for p in network.get("VpcPeeringConnections", [])
        if p.get("Status", {}).get("Code") == "active"
        and (
            (
                p.get("RequesterVpcInfo", {}).get("VpcId") in app_vpcs
                and p.get("AccepterVpcInfo", {}).get("VpcId") in files_vpcs
            )
            or (
                p.get("AccepterVpcInfo", {}).get("VpcId") in app_vpcs
                and p.get("RequesterVpcInfo", {}).get("VpcId") in files_vpcs
            )
        )
    ]
    separation = bool(app_vpcs and files_vpcs and not app_vpcs & files_vpcs)
    no_tgw = not any(
        a.get("ResourceId") in app_vpcs | files_vpcs and a.get("State") not in ("deleted", "deleting")
        for a in network.get("TransitGatewayAttachments", [])
    )
    mark(
        "A1-04",
        separation and peers and no_tgw,
        f"app VPCs={sorted(app_vpcs)}, files VPCs={sorted(files_vpcs)}, active peerings={len(peers)}",
        "containers",
        "files",
        "network",
    )
    mark(
        "A1-05",
        sleepers
        and exporters
        and all(
            128 <= f.get("MemorySize", 0) <= 1024 and 3 <= f.get("Timeout", 0) <= 120 for f in sleepers + exporters
        ),
        f"Application Lambda memory/timeouts={[(f['FunctionName'], f.get('MemorySize'), f.get('Timeout')) for f in sleepers + exporters]}",
        "functions",
    )
    origins = distribution.get("Origins", {}).get("Items", [])
    console_origins = [o for o in origins if ".s3" in o.get("DomainName", "")]
    buckets = [data.get("buckets", {}).get(o["DomainName"].split(".s3")[0], {}) for o in console_origins]
    lifecycle = bool(buckets) and all(
        any(r.get("Status") == "Enabled" for r in b.get("lifecycle", [])) for b in buckets
    )
    mark(
        "A1-06",
        lifecycle,
        f"Console buckets with active lifecycle: {sum(bool(b.get('lifecycle')) for b in buckets)}/{len(buckets)}",
        "edge",
    )
    mark(
        "B1-01",
        separation,
        f"Application VPCs={sorted(app_vpcs)}; EFS VPCs={sorted(files_vpcs)}",
        "containers",
        "files",
        "network",
    )
    nfs_rules = [
        rule
        for t in targets
        for g in t["groups"]
        for rule in groups.get(g, {}).get("IpPermissions", [])
        if rule.get("IpProtocol") == "-1"
        or rule.get("IpProtocol") == "tcp"
        and rule.get("FromPort", 0) <= 2049 <= rule.get("ToPort", 65535)
    ]
    nfs = bool(nfs_rules) and all(
        not rule.get("IpRanges")
        and not rule.get("Ipv6Ranges")
        and rule.get("UserIdGroupPairs")
        and all(groups.get(g["GroupId"], {}).get("VpcId") in app_vpcs for g in rule["UserIdGroupPairs"])
        for rule in nfs_rules
    )
    peer_ids = {p["VpcPeeringConnectionId"] for p in peers}
    routes = [
        r
        for table in network.get("RouteTables", [])
        for r in table.get("Routes", [])
        if r.get("VpcPeeringConnectionId") in peer_ids and r.get("State") == "active"
    ]
    mark(
        "B1-02",
        separation and peers and nfs and len(routes) >= 2,
        f"Peering routes={len(routes)}, NFS restricted to app security groups={nfs}",
        "network",
        "files",
    )
    keys_ok = expected <= tables.keys()
    for suffix, sort in (("users", None), ("meals", "meal_id"), ("workouts", "workout_id"), ("sleep", "session_id")):
        wanted = {"user_id": "HASH"} | ({sort: "RANGE"} if sort else {})
        keys_ok = (
            keys_ok
            and {k["AttributeName"]: k["KeyType"] for k in tables.get("vitalforge-" + suffix, {}).get("KeySchema", [])}
            == wanted
        )
    users = {u.get("user_id", {}).get("S") for u in tables.get("vitalforge-users", {}).get("users", [])}
    mark(
        "B1-03",
        keys_ok and {"member-demo-1", "member-demo-2", "member-demo-3"} <= users,
        f"Exact DynamoDB keys={keys_ok}, seed users={sorted(users - {None})}",
        "database",
    )
    mark("B1-04", running, container_detail, "containers")
    mark(
        "B1-05",
        any({"coach", "member"} <= {g["GroupName"] for g in pool} for pool in data.get("identity", [])),
        "Cognito pool must contain both coach and member groups",
        "identity",
    )
    behaviors = distribution.get("CacheBehaviors", {}).get("Items", [])
    default = distribution.get("DefaultCacheBehavior", {})
    routes_ok = (
        edge
        and default.get("TargetOriginId") in {o["Id"] for o in console_origins}
        and all(
            any(fnmatch.fnmatch(path, b.get("PathPattern", "")) for b in behaviors)
            for path in ("/api/members", "/health", "/sleep")
        )
    )
    waf = bool(distribution.get("WebACLId"))
    mark("B1-06", routes_ok and waf, f"Console/API/sleep routing={bool(routes_ok)}, associated WAF={waf}", "edge")
    config = data.get("configuration", {})
    active = json.loads(config.get("/vitalforge/thresholds/active", "{}"))
    probe = evidence.get("probe") or {}
    current_version = (probe.get("dashboard") or {}).get("analytics", {}).get("thresholds_version")
    mark(
        "B1-07",
        active.get("version") and current_version == active["version"] and passed("thresholds-version", "sleep-ingest"),
        f"SSM active={active.get('version')}, dashboard runtime={current_version}, observed versions={probe.get('versions', [])}",
        "configuration",
    )
    lattice = data.get("lattice", {})
    lattice_networks = [
        n
        for n in lattice.get("networks", [])
        if any(v.get("vpcId") in app_vpcs and v.get("status") == "ACTIVE" for v in n["vpcs"])
    ]
    linked = {s.get("serviceId") for n in lattice_networks for s in n["services"] if s.get("status") == "ACTIVE"}
    services = [s for s in lattice.get("services", []) if s["id"] in linked]
    healthy_targets = {
        t["id"]
        for g in lattice.get("groups", [])
        if set(g.get("serviceArns", [])) & {s["arn"] for s in services}
        for t in g["targets"]
        if t.get("status") == "HEALTHY"
    }
    lattice_ok = bool(services and analytics_ips and analytics_ips <= healthy_targets)
    mark(
        "B1-08",
        lattice_ok,
        f"Linked Lattice services={len(services)}, healthy analytics targets={len(analytics_ips & healthy_targets)}",
        "lattice",
        "containers",
    )
    dns_ok = any(
        z["Name"] == "vitalforge.internal."
        and any(v["VPCId"] in app_vpcs for v in z["vpcs"])
        and any(
            r["Name"] == "analytics.vitalforge.internal."
            and any(
                record["Value"].rstrip(".") == s.get("dnsEntry", {}).get("domainName", "").rstrip(".")
                for record in r.get("ResourceRecords", [])
                for s in services
            )
            for r in z["records"]
        )
        for z in lattice.get("zones", [])
    )
    mark(
        "B1-09",
        dns_ok,
        "Private analytics.vitalforge.internal must resolve to the associated Lattice service in app-vpc",
        "lattice",
    )
    api_data = data.get("api", {})
    api_origins = {o.get("DomainName", "") for o in origins}
    apis = [
        a for a in api_data.get("apis", []) if urllib.parse.urlparse(a.get("ApiEndpoint", "")).hostname in api_origins
    ]
    integrations = {i["IntegrationId"]: i for a in apis for i in a["integrations"]}
    sleep_routes = [
        r
        for a in apis
        for r in a["routes"]
        if r["RouteKey"] in ("POST /sleep", "ANY /sleep", "ANY /{proxy+}", "$default")
    ]
    sleep_api = any(
        any(
            f["FunctionArn"] in integrations.get(r.get("Target", "").split("/")[-1], {}).get("IntegrationUri", "")
            for f in sleepers
        )
        for r in sleep_routes
    )
    mark(
        "B1-10",
        sleep_api
        and passed("sleep-ingest", "sleep-too-short")
        and probe.get("sleep_missing")
        and probe.get("sleep_unknown"),
        f"Original sleep Lambda integration={sleep_api}; 202/400/422 contract and unknown-user rejection",
        "api",
        "functions",
    )
    exporter_arns = {f["FunctionArn"] for f in exporters}
    workflow = data.get("workflow", {})
    scheduled = any(
        s.get("State") == "ENABLED" and s.get("Target", {}).get("Arn") in exporter_arns
        for s in workflow.get("schedules", [])
    ) or any(
        r.get("State") == "ENABLED"
        and r.get("ScheduleExpression")
        and any(t.get("Arn") in exporter_arns for t in r.get("targets", []))
        for r in workflow.get("rules", [])
    )
    mark(
        "B1-11",
        scheduled and point_arns and probe.get("overwritten"),
        f"Scheduled original daily-export={scheduled}; EFS mounts={len(point_arns)}; observed export replacement={probe.get('overwritten', False)}",
        "workflow",
        "functions",
    )
    trails = data.get("audit", [])
    audited = [
        t
        for t in trails
        if t.get("IsMultiRegionTrail")
        and t.get("status", {}).get("IsLogging")
        and t.get("status", {}).get("LatestDeliveryTime")
        and (
            any(s.get("IncludeManagementEvents") for s in t.get("selectors", {}).get("EventSelectors", []))
            or any(
                any(
                    f.get("Field") == "eventCategory" and "Management" in f.get("Equals", [])
                    for f in s.get("FieldSelectors", [])
                )
                for s in t.get("selectors", {}).get("AdvancedEventSelectors", [])
            )
        )
    ]
    mark("B1-12", audited, f"Active management trails with successful S3 delivery={len(audited)}", "audit")
    pipelines = [
        p
        for p in data.get("pipeline", [])
        if any(
            a.get("configuration", {}).get("S3ObjectKey") == "console.zip"
            for stage in p["stages"]
            for a in stage["actions"]
        )
    ]
    deployed = any(
        p.get("state", {}).get("stageStates")
        and all(s.get("latestExecution", {}).get("status") == "Succeeded" for s in p["state"]["stageStates"])
        and any(
            "cloudfront create-invalidation" in project.get("source", {}).get("buildspec", "")
            for project in p["projects"]
        )
        for p in pipelines
    )
    mark("B1-13", deployed, f"Successful console.zip deployment and CloudFront invalidation={deployed}", "pipeline")
    api_routes = [
        r
        for a in apis
        for r in a["routes"]
        if "/api/" in r["RouteKey"] or r["RouteKey"] in ("$default", "ANY /{proxy+}")
    ]
    mark(
        "C1-01",
        api_routes
        and all(r.get("AuthorizationType", "NONE") == "NONE" for r in api_routes)
        and passed("members-unauth", "dashboard-forbidden"),
        "API Gateway forwards protected routes without JWT authorizers; application rejects anonymous/cross-member requests",
        "api",
    )
    public_groups = [g for g in api_data.get("groups", []) if g.get("LoadBalancerArns")]
    public_targets = {t["Target"]["Id"] for g in public_groups for t in g["targets"]}
    exposed = bool(analytics_ips & public_targets) or any(
        "analytics" in i.get("IntegrationUri", "").lower() for a in apis for i in a["integrations"]
    )
    mark(
        "C1-02",
        lattice_ok and not exposed,
        f"Healthy private Lattice analytics={lattice_ok}, analytics exposed through ALB/API={exposed}",
        "api",
        "lattice",
    )
    private_buckets = (
        bool(buckets)
        and all(
            all(
                b.get("public", {}).get(k)
                for k in ("BlockPublicAcls", "BlockPublicPolicy", "IgnorePublicAcls", "RestrictPublicBuckets")
            )
            for b in buckets
        )
        and all(
            o.get("OriginAccessControlId") or o.get("S3OriginConfig", {}).get("OriginAccessIdentity")
            for o in console_origins
        )
    )
    mark("C1-03", private_buckets, "All console origins require public access blocks and OAC/OAI", "edge")
    mark(
        "C1-04",
        separation and nfs,
        f"EFS isolated in files-vpc={separation}, approved SG-only NFS={nfs}",
        "files",
        "network",
    )
    mark("C1-05", waf, f"CloudFront WebACLId={distribution.get('WebACLId', '')}", "edge")
    used_accounts = {
        (d["metadata"].get("namespace", "default"), d["spec"]["template"]["spec"].get("serviceAccountName", "default"))
        for d in workloads
    }
    roles = {a["roleArn"] for a in pod_roles if (a["namespace"], a["serviceAccount"]) in used_accounts}
    roles.update(
        a.get("metadata", {}).get("annotations", {}).get("eks.amazonaws.com/role-arn")
        for a in accounts
        if (a["metadata"].get("namespace", "default"), a["metadata"]["name"]) in used_accounts
    )
    roles.discard(None)
    pod_iam = bool(roles)
    roles.update(f["Role"] for f in sleepers + exporters)
    broad = []
    for arn in roles:
        docs = data.get("policies", {}).get(arn, [])
        if not docs:
            broad.append(arn)
        for doc in docs:
            for statement in doc.get("Statement", []):
                if statement.get("Effect") != "Allow":
                    continue
                actions = statement.get("Action", [])
                actions = [actions] if isinstance(actions, str) else actions
                resources = statement.get("Resource", [])
                resources = [resources] if isinstance(resources, str) else resources
                unscoped = [
                    a
                    for a in actions
                    if not a.lower().startswith(
                        (
                            "ec2:describe",
                            "ec2:createnetworkinterface",
                            "ec2:deletenetworkinterface",
                            "ec2:assignprivateipaddresses",
                            "ec2:unassignprivateipaddresses",
                            "elasticfilesystem:describe",
                            "logs:createloggroup",
                            "xray:",
                            "ssmmessages:",
                        )
                    )
                ]
                if any("*" in a for a in actions) or "*" in resources and unscoped or statement.get("NotAction"):
                    broad.append(arn)
    mark(
        "C1-06",
        pod_iam and sleepers and exporters and not broad,
        f"Application pod/Lambda roles={len(roles)}, broad or unreadable roles={broad}",
        "containers",
        "functions",
    )
    open_ssh = [
        g["GroupId"]
        for g in groups.values()
        for rule in g.get("IpPermissions", [])
        if (
            rule.get("IpProtocol") == "-1"
            or rule.get("IpProtocol") == "tcp"
            and rule.get("FromPort", 0) <= 22 <= rule.get("ToPort", 65535)
        )
        and any(r.get("CidrIp") == "0.0.0.0/0" for r in rule.get("IpRanges", []))
    ]
    mark("C1-07", groups and not open_ssh, f"World-open SSH groups={open_ssh}", "network")
    audit_buckets = [data.get("buckets", {}).get(t["S3BucketName"], {}) for t in trails]
    mark(
        "C1-08",
        audit_buckets
        and all(
            all(
                b.get("public", {}).get(k)
                for k in ("BlockPublicAcls", "BlockPublicPolicy", "IgnorePublicAcls", "RestrictPublicBuckets")
            )
            for b in audit_buckets
        ),
        "CloudTrail destination buckets block public access",
        "audit",
    )
    mark(
        "C1-09",
        running and sleepers and exporters and deployed,
        "Original Lambda ZIP hashes match; named EKS applications run and console.zip pipeline succeeded. Container payload and deployed console bytes require manual checksum verification.",
        "functions",
        "containers",
        "pipeline",
    )
    efs_volumes = [v for v in volumes if v.get("spec", {}).get("csi", {}).get("driver") == "efs.csi.aws.com"]
    mount_secure = bool(efs_volumes) and all(
        "fsap-" in v["spec"]["csi"].get("volumeHandle", "") and {"tls", "iam"} <= set(v["spec"].get("mountOptions", []))
        for v in efs_volumes
    )
    mark(
        "C1-10",
        mount_secure and point_arns,
        f"EKS EFS access-point volumes with tls+iam={mount_secure}; Lambda access points={len(point_arns)}",
        "containers",
        "files",
        "functions",
    )
    mark(
        "C1-11",
        expected <= tables.keys()
        and not data.get("database", {}).get("rds")
        and not data.get("database", {}).get("aurora"),
        "Application data in DynamoDB; no RDS/Aurora in marking region",
        "database",
    )
    regions = {b.get("region") for b in buckets + audit_buckets}
    mark(
        "C1-12",
        running and systems and expected <= tables.keys() and regions == {REGION},
        f"Regional application APIs use {REGION}; console/audit bucket regions={sorted(regions - {None})}. IAM, CloudFront and its WAF are global.",
        "containers",
        "files",
        "database",
    )
    zones = {
        node_map.get(p["spec"].get("nodeName"), {})
        .get("metadata", {})
        .get("labels", {})
        .get("topology.kubernetes.io/zone")
        for p in app_pods
    } - {None}
    resilient = running and all(d.get("status", {}).get("availableReplicas", 0) >= 2 for d in workloads)
    mark(
        "D1-01",
        resilient or running and len(zones) >= 2,
        f"Application AZs={sorted(zones)}, available replicas={[d.get('status', {}).get('availableReplicas', 0) for d in workloads]}",
        "containers",
    )
    persisted = []
    for check, table, key in (
        ("meal-create", "vitalforge-meals", "meal_id"),
        ("workout-create", "vitalforge-workouts", "workout_id"),
        ("sleep-ingest", "vitalforge-sleep", "session_id"),
    ):
        body = evidence.get("checks", {}).get(check, {}).get("data", {})
        if body.get(key) and table in tables:
            try:
                item = fetch(
                    "dynamodb",
                    "get_item",
                    TableName=table,
                    Key={"user_id": {"S": "member-demo-1"}, key: {"S": body[key]}},
                    ConsistentRead=True,
                ).get("Item", {})
                if item and (
                    check != "sleep-ingest"
                    or item.get("thresholds_version", {}).get("S") == body.get("thresholds_version")
                ):
                    persisted.append(check)
            except (BotoCoreError, ClientError) as error:
                errors["persistence"] = str(error)[:240]
    mark(
        "D1-02",
        len(persisted) == 3,
        f"Latest acknowledged meal/workout/sleep IDs found with consistent reads={persisted}",
        "database",
        "persistence",
    )
    hints = [
        e.get("value", "")
        for d in apps["wellness-api"]
        for c in d["spec"]["template"]["spec"]["containers"]
        for e in c.get("env", [])
        if e["name"] == "ANALYTICS_LATTICE_URL"
    ]
    hints.append(config.get("/vitalforge/connections/analytics_lattice_url", ""))
    mark(
        "D1-03",
        lattice_ok
        and dns_ok
        and passed("dashboard-member")
        and any(urllib.parse.urlparse(h).hostname == "analytics.vitalforge.internal" for h in hints),
        f"Dashboard analytics via healthy Lattice and private DNS={bool(lattice_ok and dns_ok)}",
        "lattice",
        "configuration",
        "containers",
    )
    mark(
        "D1-04",
        point_arns and probe.get("overwritten") and passed("export-latest"),
        f"New generated_at observed at existing member export path={probe.get('overwritten', False)}",
        "functions",
    )
    logs = {g["logGroupName"] for g in data.get("logs", [])}
    pod_logs = any(
        "fluent" in c.get("image", "").lower() or "cloudwatch-agent" in c.get("image", "")
        for p in pods
        for c in p["spec"].get("containers", [])
    ) and any("containerinsights" in name or "vitalforge" in name and "pods" in name for name in logs)
    lambda_logs = (
        sleepers and exporters and all(f"/aws/lambda/{f['FunctionName']}" in logs for f in sleepers + exporters)
    )
    mark(
        "D1-05",
        pod_logs and lambda_logs,
        f"Pod log collector/groups={bool(pod_logs)}, application Lambda groups={bool(lambda_logs)}",
        "logs",
        "containers",
        "functions",
    )
    mark(
        "D1-06",
        systems and all(f.get("Encrypted") for f in systems),
        f"Encrypted application EFS systems={len(systems)}",
        "files",
    )
    mark(
        "D1-07",
        passed("export-queue", "export-latest") and probe.get("overwritten"),
        "Coach export accepted; subsequent reads observe regenerated export",
        "functions",
    )
    mark(
        "E1-01",
        edge
        and routes_ok
        and all(b.get("ViewerProtocolPolicy") in ("redirect-to-https", "https-only") for b in [default] + behaviors),
        f"Single submitted HTTPS entry={host}",
        "edge",
    )
    autoscaling = [
        h
        for h in hpas
        if h.get("spec", {}).get("scaleTargetRef", {}).get("name") in app_names
        and h["spec"].get("maxReplicas", 0) > h["spec"].get("minReplicas", 1)
        and h.get("status", {}).get("currentReplicas", 0) > 0
    ]
    mark(
        "E1-02",
        running and autoscaling,
        f"Active application HPAs={[h['metadata']['name'] for h in autoscaling]}",
        "containers",
    )
    cache_policy = data.get("edge", {}).get("policies", {}).get(default.get("CachePolicyId"), default)
    mark(
        "E1-03",
        edge
        and console_origins
        and default.get("TargetOriginId") in {o["Id"] for o in console_origins}
        and cache_policy.get("DefaultTTL", 0) > 0,
        f"Console cache default TTL={cache_policy.get('DefaultTTL', 0)}",
        "edge",
    )
    transitions = bool(buckets) and all(
        any(
            r.get("Status") == "Enabled"
            and any(
                t.get("StorageClass")
                in ("STANDARD_IA", "ONEZONE_IA", "GLACIER", "GLACIER_IR", "DEEP_ARCHIVE", "INTELLIGENT_TIERING")
                for t in r.get("Transitions", []) + r.get("NoncurrentVersionTransitions", [])
            )
            for r in b.get("lifecycle", [])
        )
        for b in buckets
    )
    mark("F1-01", transitions, "Console lifecycle transitions objects or versions toward IA/archive", "edge")
    scale_in = bool(autoscaling) and all(
        h.get("spec", {}).get("behavior", {}).get("scaleDown", {}).get("selectPolicy") != "Disabled"
        for h in autoscaling
    )
    mark("F1-02", running and scale_in, f"Application HPA scale-down enabled={scale_in}", "containers")
    mark(
        "F1-03",
        running
        and all(len(d) == 1 for d in apps.values())
        and len(clusters) == 1
        and len(systems) == 1
        and len(sleepers) == 1
        and len(exporters) == 1
        and edge,
        f"Application clusters={len(clusters)}, deployments={len(workloads)}, EFS={len(systems)}, sleep/export Lambdas={len(sleepers)}/{len(exporters)}",
        "containers",
        "files",
        "functions",
    )
    mark("F1-04", on_demand, "All four application tables use on-demand billing", "database")
    mark(
        "F1-05",
        exporters
        and all(
            1 <= (f.get("concurrency") if f.get("concurrency") is not None else f.get("accountConcurrency", 0)) <= 10
            for f in exporters
        ),
        f"Export reserved/account concurrency={[(f.get('concurrency'), f.get('accountConcurrency')) for f in exporters]}",
        "functions",
    )
    mark(
        "G1-01",
        len(persisted) == 3
        and lattice_ok
        and passed("dashboard-member", "export-queue", "export-latest")
        and probe.get("overwritten"),
        "Current writes persist, dashboard includes Lattice analytics, and export is regenerated/readable",
        "database",
        "lattice",
        "persistence",
    )
    mark(
        "G1-02",
        edge and probe.get("console") and passed("health", "members-coach", "sleep-ingest"),
        "Submitted CloudFront endpoint serves console, health, authenticated APIs and sleep",
    )
    return results


def handler(event, context):
    if not hmac.compare_digest(
        event.get("headers", {}).get("authorization", ""), "Bearer " + os.environ["ASSESSOR_TOKEN"]
    ):
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
