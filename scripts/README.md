# FIWARE Keystone Role Scripts

Scripts for dumping and summarizing role assignment data from a FIWARE Keystone deployment via the `fiware` CLI.

## Overview

| Script | Purpose |
|---|---|
| `dump_keystone_role_data.py` | Dump per-domain role data (users, groups, roles) to JSON files |
| `build_role_summary.py` | Summarize the dumped data and report anomalous role assignments |
| `find_empty_userroles.sh` | Find users whose role dump returned empty (`{}`), indicating a failed query |

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

Reads the JSON files produced by `dump_keystone_role_data.py` and produces a summary of role assignments. By default, it filters to show only entries where a user/group has component-specific roles (e.g., `ServiceAdminIOTAGENT`) without the corresponding generic role (e.g., `ServiceAdmin`).

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
- **stderr**: Human-readable summary listing affected users and groups per domain.

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

## Typical Workflow

```bash
# Step 1: Dump role data from Keystone
python3 dump_keystone_role_data.py --fiware-bin /opt/fiware/bin/fiware

# Step 2: Summarize and find anomalies
python3 build_role_summary.py -v
```
