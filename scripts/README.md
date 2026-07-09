# FIWARE Keystone Role Scripts

Scripts for dumping and summarizing role assignment data from a FIWARE Keystone deployment via the `fiware` CLI.

## Overview

| Script | Purpose |
|---|---|
| `dump_keystone_role_data.py` | Dump per-domain role data (users, groups, roles) to JSON files |
| `build_role_summary.py` | Summarize the dumped data and report anomalous role assignments |
| `find_empty_userroles.sh` | Find users whose role dump returned empty (`{}`), indicating a failed query |
| `requery_empty_userroles.py` | Re-run the `userroles` query only for users whose dumped role document is empty |

## dump_keystone_role_data.py

Queries the FIWARE Keystone instance for all domains and their role assignments, writing the results as JSON files under a `domains/` directory.

### Usage

```bash
python3 dump_keystone_role_data.py [options]
```

### Options

| Option | Description | Default |
|---|---|---|
| `--fiware-bin PATH` | Path to the fiware CLI binary | `fiware` |
| `--outdir DIR` | Directory where domain data is stored | `domains` |
| `--include-admin-domain` | Also dump the `admin_domain` | Excluded by default |
| `--force` | Redownload files even if they already exist | Skip existing files |
| `--continue-on-error` | Continue processing remaining domains on failure | Exit on first error |

### Output Structure

```
domains/
├── _domains.json              # List of all domains
├── <domain-name>/
│   ├── rolemap.json           # Users and groups in the domain
│   ├── userroles_<user-id>.json  # Role assignments per user
│   └── grouproles_<group-id>.json # Role assignments per group
```

## build_role_summary.py

Reads the JSON files produced by `dump_keystone_role_data.py` and produces a summary of role assignments. By default, it filters to show only entries where a user/group has component-specific roles (e.g., `ServiceAdminIOTAGENT`) without the corresponding generic role (e.g., `ServiceAdmin`). For flagged user entries, it also reports the `role_id`s to use for remediation later.

### Usage

```bash
python3 build_role_summary.py [options]
```

### Options

| Option | Description | Default |
|---|---|---|
| `-v, --verbose` | Print all mappings without filtering | Filtered output |

### Output

- **stdout**: JSON object mapping each domain to its `user_roles` and `group_scopes`.
- **stderr**: Human-readable summary listing affected users and groups per domain, followed by remediation `curl` commands.

### User Output Format

For each filtered user entry, the output now includes:

- `roles`: the roles currently assigned in that scope
- `missing_role_names`: the missing generic role names inferred from the current assignments
- `missing_role_ids`: the `role_id`s to use for remediation
- `user_id`: the affected user id
- `domain_id`: the domain id for domain-scoped assignments
- `project_id`: the project id for project-scoped assignments

The script checks missing generic role families in this order:

1. `ServiceAdmin`
2. `ServiceCustomer`
3. `SubServiceAdmin`
4. `SubServiceCustomer`

For remediation, the script uses these role IDs:

- missing `ServiceAdmin` -> use the `SubServiceAdmin` role ID
- missing `ServiceCustomer` -> use the `SubServiceCustomer` role ID
- missing `SubServiceAdmin` -> use the `SubServiceAdmin` role ID
- missing `SubServiceCustomer` -> use the `SubServiceCustomer` role ID

This special case exists because `ServiceAdmin` and `ServiceCustomer` are domain-level or inherited-domain roles, while the planned remediation targets project assignments.

After the summary, the script also prints remediation commands in this form:

```bash
curl -X PUT -H "X-Auth-Token: $TOKEN" \
  "https://auth.iotplatform.telefonica.com:15001/v3/projects/<project_id>/users/<user_id>/roles/<role_id>"
```

Use `--keystone-url` to override the Keystone base URL. If omitted, the script uses `KEYSTONE_URL` from the environment when present, otherwise `https://auth.iotplatform.telefonica.com:15001`.

Example:

```json
{
  "jcyl": {
    "user_roles": {
      "josefernandez|some-scope": {
        "user_id": "0b5aaafccd1c4657a9f6e0c8a4e8d148",
        "project_id": "7e5f656adb30408fa5a53a6d82c0ab6a",
        "roles": [
          "SubServiceAdminORION"
        ],
        "missing_role_names": [
          "SubServiceAdmin"
        ],
        "missing_role_ids": [
          "c1177c373578480a81bc8a07518e09a6"
        ]
      }
    },
    "group_scopes": {}
  }
}
```

### Filtering Logic

An entry is flagged when a user or group has a component-specific role without the matching generic role:

| Generic Role | Component-Specific Roles Checked |
|---|---|
| `admin`, `ServiceAdmin` | `ServiceAdminIOTAGENT`, `ServiceAdminORION`, `ServiceAdminPERSEO`, `ServiceAdminSTH` |
| `ServiceCustomer` | `ServiceCustomerIOTAGENT`, `ServiceCustomerORION`, `ServiceCustomerPERSEO`, `ServiceCustomerSTH` |
| `SubServiceAdmin` | `SubServiceAdminIOTAGENT`, `SubServiceAdminORION`, `SubServiceAdminPERSEO`, `SubServiceAdminSTH` |
| `SubServiceCustomer` | `SubServiceCustomerIOTAGENT`, `SubServiceCustomerORION`, `SubServiceCustomerPERSEO`, `SubServiceCustomerSTH` |

## find_empty_userroles.sh

Finds users whose `userroles_<id>.json` file contains only `{}`, meaning the role query failed during dump. Looks up each user's name from `rolemap.json` and prints the domain folder, username, and user ID.

### Usage

```bash
bash find_empty_userroles.sh
```

The script searches under `/home/rafa/projects/fiware/scripts/domains`. Update the path in the script if your data lives elsewhere.

### Output

Each line shows a domain and user with an empty role dump:

```
domain-name: username (user ID: abc123)
```

## requery_empty_userroles.py

Scans the dumped `domains/` data for `userroles_*.json` files containing only `{}` and re-runs the corresponding `fiware get --domain <domain> --userid <id> userroles` query. Successful responses overwrite the empty JSON files in place. Failed queries are collected into a JSON summary, including generic missing-resource fields plus type-specific fields such as `missing_project_id` or `missing_domain_id` when Keystone returns `Could not find ...` errors.

### Usage

```bash
python3 requery_empty_userroles.py [options]
```

### Options

| Option | Description | Default |
|---|---|---|
| `--fiware-bin PATH` | Path to the fiware CLI binary | `fiware` |
| `--domains-dir DIR` | Directory containing dumped domain data | `scripts/domains` |
| `--dry-run` | Print the users that would be queried without rewriting files | Disabled |
| `--failure-summary PATH` | Path to write the JSON summary of failed re-queries | `scripts/requery_empty_userroles_failures.json` |
| `--normalize-failure-summary PATH` | Reparse an existing failure summary and enrich structured error fields without querying FIWARE | Disabled |

### Failure Summary Format

```json
{
  "processed": 40,
  "updated": 12,
  "failures": 28,
  "services": {
    "jcyl": [
      {
        "user_name": "ecovidrio",
        "user_id": "6ea905c60e0641528d8f6a75e921cd4e",
        "missing_project_id": "7e5f656adb30408fa5a53a6d82c0ab6a",
        "missing_domain_id": null,
        "missing_resource_type": "project",
        "missing_resource_id": "7e5f656adb30408fa5a53a6d82c0ab6a",
        "http_status": "404",
        "error": "..."
      }
    ]
  }
}
```

## Cleanup Notes

During analysis of empty `userroles_*.json` dumps, the root cause turned out to be orphaned rows in Keystone's `assignment` table. The failing users had role assignments pointing to projects or domains that no longer existed, which caused Keystone to return `404 Could not find project` or `404 Could not find domain` during `role_assignments` queries.

The cleanup was performed directly in the Keystone database inside a transaction:

```sql
BEGIN;

SELECT count(*) AS rows_to_delete
FROM keystone.assignment a
LEFT JOIN keystone.project p
  ON p.id = a.target_id
WHERE a.type = 'UserProject'
  AND p.id IS NULL;

DELETE FROM keystone.assignment a
WHERE a.type = 'UserProject'
  AND NOT EXISTS (
    SELECT 1
    FROM keystone.project p
    WHERE p.id = a.target_id
  );

SELECT count(*) AS orphan_userdomain_rows
FROM keystone.assignment a
LEFT JOIN keystone.project d
  ON d.id = a.target_id
 AND d.is_domain = true
WHERE a.type = 'UserDomain'
  AND d.id IS NULL;

DELETE FROM keystone.assignment
WHERE type = 'UserDomain'
  AND actor_id = '4d7137dd455e4eeebf5d53e7ca0d3471'
  AND target_id = '3ec102ac4ad448a2a3d151fcb5609fd0'
  AND role_id = '8634f016f86e45f28f5d2b35a43418d3'
  AND inherited = false;

COMMIT;
```

In the observed cleanup, this removed:

- `42` orphaned `UserProject` rows
- `1` orphaned `UserDomain` row

After the database cleanup, rerunning `python3 scripts/requery_empty_userroles.py` successfully refreshed all previously empty user role dumps.

## Typical Workflow

```bash
# Step 1: Dump role data from Keystone
python3 dump_keystone_role_data.py --fiware-bin /opt/fiware/bin/fiware

# Step 2: Summarize and find anomalies
python3 build_role_summary.py -v
```
