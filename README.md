# CloudJam

> [!CAUTION]  
> Project is currently under construction 🏗️

Platform to design, host and play CTF like challenges for DevOps guys.

## NoAI Policy

CloudJam is a platform dedicated to helping people develop skills through competitive learning.

While LLMs can be useful under certain circumstances, relying on them often prevents genuine understanding of underlying concepts.

To lead by example, **we do not accept AI-generated code in the core CloudJam system**, regardless of how "superior" the quality might be. 

We strive to design our challenge SDKs with human developers in mind and strongly encourage writing challenge plugins independently. 
However, the decision to use LLM-assisted tools or services when developing your own challenge plugins is ultimately yours.

**Security audits are explicitly exempt from this policy.** AI-assisted security analysis is permitted, as the security and integrity of the platform take precedence over these restrictions.


## Deployment 🪖

> [!Warning]
> Do *NOT* use the `docker-compose.yaml` file for deployment. It is *ONLY* for development and contains hardcoded dummy secrets.


There are two ways to host CloudJam:

### hornet 🐝 

Single binary launcher used for development and cheap hosting.

**Deployment**:

```bash
TBD
```

### orca 🫍

Serverless pulumi deployment on Lambda used for cost effective scaling.

**Deployment**:

```bash
TBD
```

## Development 🚀

```bash
# starts development rustfs s3 server including webui on http://127.0.0.1:9001 ("cloudjam-access-key" & "cloudjam-secret-key").
docker-compose up -d

# build webapp initially and start the local sveltekit vite server
cd web && pnpm i && pnpm run build && pnpm run dev

# in another terminal start hornet in dev mode (s3 params default to the hardcoded options in the docker-compose)
go run ./cmd/hornet -D --token-secret 123

# use the default addr for development; hornet proxies the ui to vite so don't worry about that. 
xdg-open http://127.0.0.1:8000
```

## RBAC Concept 🏷️

CloudJam uses a **one-role-per-user** RBAC concept (a user cannot be attached to multiple roles at the same time). 
This avoids ambiguity with multisource- or transitive permissions.


The permission model is based on a *zerotrust* security concept with two layers:

1. **Data permissions**: (defines which resources can be accessed) implemented by having a "scope" on every resource. Database requests MUST be checked against the granted "scopes".
2. **Action permissions**: (defines which rpc actions can be invoked) implemented by validating the rpc procedure name against a list of basic glob patterns defined per scope in the role.

**To visualize 🔮**: 

Every rpc call goes through `map[scope][]actionExpr` returning all scopes with at least 1 matching actionExpr.
If at least 1 scope matches the action is executed with data permissions being restricted to the matched scopes.


> [!IMPORTANT]  
> There are two special builtin scopes that may be attached to a role:
>
> **ScopeSelf** ("self"): Allows access to user owned data regardless of resource scope (-> self management). 
>
> **ScopeAdmin** ("admin"): Allows privilege escalation in the `rbac.ConfigureRole` call (-> full root access).


To keep critical code together and reducing dangerous misconfigurations the `rbac` API isolates all operations which can escalate privileges (e.g. `AttachRole`). Other APIs like `user` and `role` only manage metadata and should never even be capable of escalating privileges.

> [!NOTE]  
> Action permissions are ENFORCED via middleware, data permissions are be implemented manually per call.
> For OLAP data there is generally no data permission.


**Features 🧩**:

Besides the core permission layer, every role specifies a list of **features**.
Features provide a user-friendly *"pseudo-security"* layer on top of the actual security to hide things the user has no access to instead of aggressively throwing around 403 errors (yes, AWS, I'm looking at you...)
