[← Configuration](3-Configuration.md) | Usage | [Advanced Usage →](5-Advanced.md)

---

## Usage

Basic command format:

```shell
bp <service> <action> [--Param value ...] [--header Name=Value ...] [--body json]
                      [--profile name] [--region region] [--endpoint endpoint]
                      [--version api-version] [--method GET|POST] [--force]
                      [--output json|table|table-num|text|yaml|off] [--query jmespath]
```

Argument kinds:

- **API parameters**: double-dash `--Param value` (enter request body/query; reserved names `body` / `header` excluded)
- **Public system flags** (after the action): `--profile` / `--region` / `--endpoint` / `--version` / `--method` / `--force` / `--output` / `--query`
- **Reserved double-dash controls**: `--header` (HTTP headers), `--body` (JSON body); **not** API parameters

System flags in API calls use double hyphens and are placed after the action. If an action exposes an exact-name API parameter (case-sensitive), the double-dash form is parsed as the API parameter.

### Flag Prefix Contract

The CLI has exactly one public flag prefix: **double dash `--name`**. System flags, API parameters and the reserved controls (`--header` / `--body`) all use double dashes.

- Help output, shell completion, error messages and documentation examples show the `--name` form only.
- Conflicts are case-sensitive and resolved against the parameters the current action actually exposes: differently cased names such as `--Region` or `--Endpoint` are always API parameters.
- When an action exposes an API parameter with the **exact** name of a system flag, the double-dash form after that action is parsed as the API parameter; use an equivalent non-flag route for the system behaviour (see Known Name Conflicts).

## Discover Services and Actions

List supported services:

```shell
bp --help
```

List actions under a service:

```shell
bp ecs --help
```

Show action parameters:

```shell
bp ecs DescribeInstances --help
```

By default, `-h` / `--help` uses concise mode and shows parameter names, types, and required status without loading the full parameter corpus. Use detail mode to include descriptions and examples:

```shell
bp ecs DescribeInstances -h --detail
bp ecs DescribeInstances --help --detail
```

Using `--detail` by itself does not trigger help.

Show version:

```shell
bp version
bp -v
```

## Call APIs

Call without parameters:

```shell
bp sts GetCallerIdentity
```

Call with parameters:

```shell
bp ecs DescribeInstances --InstanceIds.1 i-1234567890abcdef0
```

Multiple parameters:

```shell
bp rds_mysql ListDBInstanceIPLists --InstanceId mysql-xxxxxx --GroupName default
```

Parameter names and values are separated by spaces. The supported syntax is:

```shell
--Param value
--region ap-southeast-1
```

Do not use `--Param=value` or `--region=ap-southeast-1`. Flag names and values must be separated by a space.

## CLI System Flags

Public system flags use the standard double-hyphen form:

| Flag | Purpose |
| --- | --- |
| `--profile` | Use a specific profile for this invocation without changing current |
| `--region` | Override region for this invocation |
| `--endpoint` | Override endpoint for this invocation and clear endpoint resolver |
| `--version` | Set the **API version** for this call; if omitted, uses the bundled service version (not the CLI binary version from root `bp -v` / `bp --version` / `bp version`) |
| `--force` | Skip service/action metadata validation and force-call unlisted or newly released APIs; **unlisted services** require `--version` and a fixed endpoint (`--endpoint` or profile/`BYTEPLUS_ENDPOINT` when resolver is not `standard`); bundled services can fall back to metadata. Presence-only: write `--force` alone, not `--force true` |
| `--method` | HTTP method (`GET`/`POST`); same rules on normal and `--force` paths: explicit value wins, else action metadata, else `GET` |
| `--output` | API response format: `json` (default), `table`, `table-num`, `text`, `yaml`, or `off` |
| `--query` | JMESPath expression applied to the full response before formatting |

After the action, a double-dash flag whose exact case-sensitive name is exposed by that action is parsed as an API parameter. Without such a conflict, it is parsed as a system flag.

Names with different casing, such as `--Region` or `--Endpoint`, are always API parameters.


### Known Name Conflicts

Published BytePlus metadata currently exposes **no** API parameter whose exact case-sensitive name matches a public system flag, so every system flag keeps its system meaning after any bundled action.

The rule still applies if a future metadata release introduces a colliding name: after that action the double-dash form is the API parameter, and the system behaviour has to be reached another way. What is available depends on the flag:

| Shadowed flag | Alternative route |
| --- | --- |
| `--profile` | `bp configure profile --profile <name>` to switch the active profile first, or the `BYTEPLUS_PROFILE` / `BYTEPLUS_CLI_PROFILE` environment variables |
| `--region` | `region` in the profile, or `BYTEPLUS_REGION` |
| `--endpoint` | `endpoint` in the profile, or `BYTEPLUS_ENDPOINT` |
| `--output`, `--query` | keep the default JSON response and filter it downstream (for example with `jq`) |
| `--version`, `--method`, `--force` | no equivalent route: these have no profile field and no environment variable, so a collision on one of them would leave the behaviour unreachable through public syntax. Report it so the flag set can be revised. |

For an unlisted action, metadata cannot declare a collision, so all public system flags retain their system meanings.

### Reserved Double-Dash Controls

| Flag | Purpose |
| --- | --- |
| `--header Name=Value` | Add an HTTP request header; **repeatable**; never enters the request body. `Content-Type` overrides metadata; last value wins for the same name |
| `--body json` | JSON request body for `application/json` style calls; mutually exclusive with other API parameters |

```shell
bp sts GetCallerIdentity --header X-Custom-Trace=abc
bp newsvc Act --force --version 2024-01-01 --endpoint open.byteplusapi.com \
  --header Content-Type=application/json \
  --header X-Feature=on \
  --body '{"k":1}'
```

Notes:

- Override `Content-Type` with `--header Content-Type=...`; forms with parameters (e.g. `application/json; charset=utf-8`) are still treated as JSON
- With `--body` and no metadata, Content-Type defaults to `application/json`
- `--header` can be used with `--body`; headers are not flattened API params and do not conflict with `--body`
- Blocked header names: `Host`, `Authorization`, `Content-Length` (transport/signing)
- Reserved names: `--header` and `--body` cannot be used as ordinary API parameter names

Examples:

```shell
# Use a specific profile
bp ecs DescribeInstances --profile prod

# Use a specific profile and override region
bp ecs DescribeInstances --profile prod --region ap-southeast-1

# Override only region
bp ecs DescribeInstances --region cn-shanghai

# Specify endpoint for an STS call
bp sts GetCallerIdentity --region ap-southeast-1 --endpoint sts.byteplusapi.com
```

If `--profile` references a profile that does not exist, the command returns an error.

## JSON Parameters

For query/form APIs, if a parameter value is a JSON object or JSON array, the CLI attempts to parse it as JSON:

```shell
bp rds_mysql ModifyDBInstanceIPList \
  --InstanceId mysql-xxxxxx \
  --GroupName default \
  --IPList '["10.20.30.40","50.60.70.80"]'
```

String parameters are kept as strings and are not forcibly parsed just because they look like JSON.

## application/json Requests

For APIs whose `ContentType` is `application/json`, pass a JSON body directly:

```shell
bp rds_mysql ModifyDBInstanceIPList \
  --body '{"InstanceId":"mysql-xxxxxx","GroupName":"default","IPList":["10.20.30.40","50.60.70.80"]}'
```

`--body` must be a JSON object or JSON array. It cannot be mixed with flattened parameters:

```shell
# Wrong: --body cannot be used together with other API parameters
bp rds_mysql ModifyDBInstanceIPList --body '{"InstanceId":"mysql-xxxxxx"}' --GroupName default
```

application/json APIs also support dotted keys. The CLI expands them into nested JSON using metadata:

```shell
bp some_service SomeJsonAction \
  --Name demo \
  --Ports.1 80 \
  --Ports.2 443 \
  --Tags.1.Key env \
  --Tags.1.Value prod
```

Array indices are 1-based and must be contiguous. `0`, negative indices, and skipped indices are errors.

## Arrays and Nested Parameters

Common array syntax:

```shell
bp ecs DescribeInstances --InstanceIds.1 i-123 --InstanceIds.2 i-456
```

Array of objects:

```shell
bp some_service SomeAction \
  --Filters.1.Key InstanceType \
  --Filters.1.Values.1 ecs.g1.large \
  --Filters.1.Values.2 ecs.g2.large
```

For application/json APIs, dotted keys are restored to nested objects and arrays. For non-JSON APIs, dotted keys are preserved and handled by the service/API layer.

## Unknown Parameters

The CLI allows unknown API parameters to pass through to the service/API layer. Unless the parameter path itself is invalid, the CLI does not reject a parameter only because it is absent from metadata.

Example:

```shell
bp ecs DescribeInstances --NewServerSideParam value
```

This is useful when the service has added a parameter but local metadata has not been updated yet.

## Unlisted Services and Actions

The CLI validates services and actions against built-in metadata. If the **service or action is not yet bundled**, use `--force` to bypass validation; unlisted services also require `--version` and a **fixed** endpoint (`--endpoint`, or profile / `BYTEPLUS_ENDPOINT` when `endpoint-resolver` is not `standard`) because the CLI has no metadata from which to resolve a host. Bundled services can omit these overrides in force mode and use metadata with the same endpoint rules as normal calls. See [Advanced Usage: Force Invocation](5-Advanced.md#force-invocation).

```shell
bp newservice DescribeNewResource \
  --version 2024-01-01 \
  --endpoint open.byteplusapi.com \
  --SomeParam value \
  --force
```

## Common Scenarios

Use current profile:

```shell
bp ecs DescribeInstances
```

Use a non-current profile:

```shell
bp ecs DescribeInstances --profile prod
```

Use environment-based default credential chain:

```shell
export BYTEPLUS_ACCESS_KEY=AK
export BYTEPLUS_SECRET_KEY=SK
export BYTEPLUS_REGION=ap-southeast-1
bp ecs DescribeInstances
```

Use an OIDC profile:

```shell
bp configure set --profile ci-oidc --mode oidc --region ap-southeast-1 \
  --oidc-token-file /var/run/secrets/oidc-token \
  --role-trn trn:iam::2100000000:role/CIRole

bp ecs DescribeInstances --profile ci-oidc
```

Use an ECS instance role profile:

```shell
bp configure set --profile ecs-role --mode ecsrole --region ap-southeast-1 --role-name MyRole
bp ecs DescribeInstances --profile ecs-role
```

## Common Errors

Missing credentials:

```text
credentials not configured, please run 'bp login' or 'bp configure set', or set BYTEPLUS_ACCESS_KEY and BYTEPLUS_SECRET_KEY environment variables
```

Missing region:

```text
region not set, please set it via profile, --region flag, or BYTEPLUS_REGION environment variable
```

Public system flags (double-dash): `--profile`, `--region`, `--endpoint`, `--force`, `--version`, `--method`, `--output`, `--query`.
Reserved double-dash controls: `--header`, `--body` (see “Reserved Double-Dash Controls” above).

## Filtering and Output Formats

A successful API call prints the full response, normally `ResponseMetadata` plus `Result`, as JSON by default. The response pipeline is `raw response → --query → --output → stdout`. Both flags apply only to the current invocation and are never persisted.

```shell
# Project selected fields, preserving the hash key order as table columns
bp ecs DescribeInstances \
  --query "Result.Instances[*].{Name:InstanceName,Id:InstanceId,Status:Status}" \
  --output table

# Add a leading row-number column
bp ecs DescribeInstances \
  --query "Result.Instances[*].{Name:InstanceName,Id:InstanceId}" \
  --output table-num

# Tab-separated text, YAML, or no response output
bp sts GetCallerIdentity --query "Result.AccountId" --output text
bp sts GetCallerIdentity --output yaml
bp ecs DescribeInstances --output off
```

Output behavior:

- `json` preserves exact response number tokens and supports optional ANSI token coloring.
- `table` and `table-num` render nested data in titled sections instead of hiding it or embedding unreadable JSON. `table-num` adds a `#` column starting at 1.
- A one-record table stays horizontal unless a known terminal width requires vertical `Field | Value` layout. Over-wide terminal tables wrap the widest columns; piped or redirected output is not width-fitted.
- `text` recursively flattens objects and lists into tab-separated rows with stable uppercase field-path labels. Nested control characters are escaped so response data cannot alter row or column boundaries.
- `yaml` preserves large integers, long decimals, exponent spelling, and trailing zeros without conversion to rounded floating-point values. YAML keys are sorted alphabetically.
- `off` still sends the API request but writes no response body and skips response-dependent query evaluation. Query syntax is still validated before the request.
- Every renderer receives the complete selected value. `ResponseMetadata`, including request identifiers, is not silently removed. Empty lists render as `(empty)` in tables and produce no text rows; a missing or null query result renders as `None` in table and text.
- For `table`, `table-num`, and `text`, an explicit JMESPath multiselect hash controls column order when its keys exactly match the rendered object. Other object keys use deterministic alphabetical order.
- Booleans are `True` / `False` in human-readable table and text formats. JSON and YAML use their native lowercase spelling.
- When color is enabled, ANSI styling is emitted only for terminal output. Redirects, pipes, and a non-empty `NO_COLOR` environment variable disable table colors. `NO_COLOR` also disables JSON colors.
- Rendering is buffered and write or flush failures are returned as command errors. API failures are written to stderr and do not pass through `--query` or `--output`.

Query behavior:

- `--query` uses JMESPath against the full response, so service data paths usually start with `Result.`. Use `--query "@"` to explicitly select the complete response.
- Syntax errors, unknown functions, wrong argument counts, incomplete indexes or expressions, and unsafe exact-number arithmetic are rejected before the API request. Diagnostics include the expression, a caret, and an actionable hint when available.
- Evaluation errors that depend on response types are reported after a successful API call as response-output failures and do not panic.
- Numeric comparisons, filters, sorting, and supported arithmetic use exact JSON decimals. Integers above 2^53 are not rounded, equivalent spellings such as `1`, `1.0`, and `1e0` compare equal, and projections retain the original token. Arithmetic with a decimal exponent above 10000 is rejected rather than silently rounded.

No bundled action currently exposes an exact API parameter named `query` or `output`, so both keep their system meanings everywhere. If a future metadata release introduces such a name, the double-dash form after that action is routed to the API parameter under the standard conflict rule (see Known Name Conflicts); for unlisted actions metadata cannot declare a collision, so both names always retain their system-flag meanings.

---

[← Configuration](3-Configuration.md) | Usage | [Advanced Usage →](5-Advanced.md)
