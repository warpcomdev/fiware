#!/usr/bin/env python3

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any, TypedDict


DOMAINS_DIR = Path(__file__).parent / "domains"
DEFAULT_KEYSTONE_URL = os.environ.get(
    "KEYSTONE_URL", "https://auth.keystone.com"
)

ADMIN_SPECIFIC = {"ServiceAdmin" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")}
CUSTOMER_SPECIFIC = {
    "ServiceCustomer" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")
}
SUB_ADMIN_SPECIFIC = {
    "SubServiceAdmin" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")
}
SUB_CUSTOMER_SPECIFIC = {
    "SubServiceCustomer" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")
}

GENERIC_ADMIN = {"ServiceAdmin", "admin"}
GENERIC_CUSTOMER = {"ServiceCustomer"}
GENERIC_SUB_ADMIN = {"SubServiceAdmin"}
GENERIC_SUB_CUSTOMER = {"SubServiceCustomer"}

REMEDIATION_ROLE_NAME = {
    "ServiceAdmin": "SubServiceAdmin",
    "ServiceCustomer": "SubServiceCustomer",
    "SubServiceAdmin": "SubServiceAdmin",
    "SubServiceCustomer": "SubServiceCustomer",
}


class UserRoleEntry(TypedDict, total=False):
    roles: list[str]
    missing_role_names: list[str]
    missing_role_ids: list[str]
    user_id: str
    domain_id: str
    project_id: str


UserScopeKey = tuple[str, str]
UserRoles = dict[UserScopeKey, UserRoleEntry]
GroupScopes = dict[UserScopeKey, list[str]]
RoleNameToID = dict[str, str]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Summarize Keystone role assignments per domain."
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Print all mappings without filtering.",
    )
    parser.add_argument(
        "--keystone-url",
        default=DEFAULT_KEYSTONE_URL,
        help="Base Keystone URL used to build remediation curl commands. Default: %(default)s",
    )
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def process_domain(domain_dir: Path) -> tuple[UserRoles, GroupScopes, RoleNameToID]:
    rolemap_path = domain_dir / "rolemap.json"
    if not rolemap_path.exists():
        return {}, {}, {}

    rolemap = load_json(rolemap_path)

    user_id_to_name: dict[str, str] = {}
    for u in rolemap.get("users", []):
        uid = u.get("id")
        uname = u.get("name")
        if uid and uname:
            user_id_to_name[uid] = uname

    group_id_to_name: dict[str, str] = {}
    for g in rolemap.get("groups", []):
        gid = g.get("id")
        gname = g.get("name")
        if gid and gname:
            group_id_to_name[gid] = gname

    role_name_to_id: dict[str, str] = {}
    for role in rolemap.get("roles", []):
        role_id = role.get("id")
        role_name = role.get("name")
        if role_id and role_name and role_name not in role_name_to_id:
            role_name_to_id[role_name] = role_id

    user_roles: UserRoles = {}
    group_scopes: GroupScopes = {}

    for fpath in sorted(domain_dir.glob("userroles_*.json")):
        data = load_json(fpath)
        for assign in data.get("assignments", []):
            user_info = assign.get("user", {})
            if not isinstance(user_info, dict):
                continue

            uname = user_info.get("name")
            if not uname:
                uid = user_info.get("id", "")
                uname = user_id_to_name.get(uid, uid)

            scope_name = assign.get("scope_name", "")
            role_name = assign.get("role", {}).get("name", "")
            if not isinstance(uname, str) or not isinstance(scope_name, str):
                continue
            if not isinstance(role_name, str) or not scope_name or not role_name:
                continue

            key = (uname, scope_name)
            if key not in user_roles:
                entry: UserRoleEntry = {"roles": []}
                if isinstance(user_info.get("id"), str):
                    entry["user_id"] = user_info["id"]
                if isinstance(assign.get("domain_id"), str):
                    entry["domain_id"] = assign["domain_id"]
                if isinstance(assign.get("project_id"), str):
                    entry["project_id"] = assign["project_id"]
                user_roles[key] = entry
            entry = user_roles[key]
            roles = entry.setdefault("roles", [])
            if role_name not in roles:
                roles.append(role_name)
            if not entry.get("user_id") and isinstance(user_info.get("id"), str):
                entry["user_id"] = user_info["id"]
            if not entry.get("domain_id") and isinstance(assign.get("domain_id"), str):
                entry["domain_id"] = assign["domain_id"]
            if not entry.get("project_id") and isinstance(
                assign.get("project_id"), str
            ):
                entry["project_id"] = assign["project_id"]

    for fpath in sorted(domain_dir.glob("grouproles_*.json")):
        data = load_json(fpath)
        for assign in data.get("assignments", []):
            group_info = assign.get("group", {})
            if not isinstance(group_info, dict):
                continue

            gname = group_info.get("name")
            if not gname:
                gid = group_info.get("id", "")
                gname = group_id_to_name.get(gid, gid)

            scope_name = assign.get("scope_name", "")
            role_name = assign.get("role", {}).get("name", "")
            if not isinstance(gname, str) or not isinstance(scope_name, str):
                continue
            if not isinstance(role_name, str) or not scope_name or not role_name:
                continue

            key = (gname, role_name)
            if key not in group_scopes:
                group_scopes[key] = []
            if scope_name not in group_scopes[key]:
                group_scopes[key].append(scope_name)

    return user_roles, group_scopes, role_name_to_id


def _missing_generic_role_names(roles: list[str]) -> list[str]:
    role_set = set(roles)
    missing_roles: list[str] = []
    if (role_set & ADMIN_SPECIFIC) and not (role_set & GENERIC_ADMIN):
        missing_roles.append("ServiceAdmin")
    if (role_set & CUSTOMER_SPECIFIC) and not (role_set & GENERIC_CUSTOMER):
        missing_roles.append("ServiceCustomer")
    if (role_set & SUB_ADMIN_SPECIFIC) and not (role_set & GENERIC_SUB_ADMIN):
        missing_roles.append("SubServiceAdmin")
    if (role_set & SUB_CUSTOMER_SPECIFIC) and not (role_set & GENERIC_SUB_CUSTOMER):
        missing_roles.append("SubServiceCustomer")
    return missing_roles


def _matches_filter(roles: list[str]) -> bool:
    return bool(_missing_generic_role_names(roles))


def _tuple_key_dict(
    d: UserRoles | GroupScopes, verbose: bool, role_name_to_id: RoleNameToID
) -> dict[str, UserRoleEntry]:
    result: dict[str, UserRoleEntry] = {}
    for k, v in d.items():
        roles = v.get("roles", []) if isinstance(v, dict) else v
        missing_role_names = _missing_generic_role_names(roles)
        if verbose or missing_role_names:
            entry: UserRoleEntry = {
                "roles": roles,
                "missing_role_names": missing_role_names,
                "missing_role_ids": [
                    role_name_to_id[REMEDIATION_ROLE_NAME[role_name]]
                    for role_name in missing_role_names
                    if REMEDIATION_ROLE_NAME[role_name] in role_name_to_id
                ],
            }
            if isinstance(v, dict):
                user_id = v.get("user_id")
                domain_id = v.get("domain_id")
                project_id = v.get("project_id")
                if user_id:
                    entry["user_id"] = user_id
                if domain_id:
                    entry["domain_id"] = domain_id
                if project_id:
                    entry["project_id"] = project_id
            result[f"{k[0]}|{k[1]}"] = entry
    return result


def _build_remediation_commands(
    results: dict[str, dict[str, dict[str, UserRoleEntry]]], keystone_url: str
) -> list[str]:
    commands: list[str] = []
    seen: set[tuple[str, str, str]] = set()
    base_url = keystone_url.rstrip("/")

    for domain_name in sorted(results):
        user_roles = results[domain_name].get("user_roles", {})
        for key in sorted(user_roles):
            entry = user_roles[key]
            user_id = entry.get("user_id")
            project_id = entry.get("project_id")
            missing_role_ids = entry.get("missing_role_ids", [])
            missing_role_names = entry.get("missing_role_names", [])

            if not user_id or not project_id:
                continue

            for idx, role_id in enumerate(missing_role_ids):
                grant_key = (project_id, user_id, role_id)
                if grant_key in seen:
                    continue
                seen.add(grant_key)

                role_name = (
                    missing_role_names[idx] if idx < len(missing_role_names) else ""
                )
                commands.append(
                    (
                        f"# {domain_name} {key} missing {role_name}\n"
                        f'curl -X PUT -H "X-Auth-Token: $TOKEN" '
                        f'"{base_url}/v3/projects/{project_id}/users/{user_id}/roles/{role_id}"'
                    )
                )

    return commands


def main() -> None:
    args = parse_args()
    results: dict[str, dict[str, dict[str, UserRoleEntry]]] = {}

    for domain_dir in sorted(DOMAINS_DIR.iterdir()):
        if not domain_dir.is_dir():
            continue

        user_roles, group_scopes, role_name_to_id = process_domain(domain_dir)
        filtered_user = _tuple_key_dict(user_roles, args.verbose, role_name_to_id)
        filtered_group = _tuple_key_dict(group_scopes, args.verbose, {})
        if filtered_user or filtered_group:
            results[domain_dir.name] = {
                "user_roles": filtered_user,
                "group_scopes": filtered_group,
            }

    print(json.dumps(results, indent=2, ensure_ascii=False))

    print("\n--- Summary ---", file=sys.stderr)
    for domain_name in sorted(results):
        users = sorted({k.split("|")[0] for k in results[domain_name]["user_roles"]})
        groups = sorted({k.split("|")[0] for k in results[domain_name]["group_scopes"]})
        print(
            f"[{domain_name}] {len(users)} users, {len(groups)} groups", file=sys.stderr
        )
        for u in users:
            print(f"  user: {u}", file=sys.stderr)
        for g in groups:
            print(f"  group: {g}", file=sys.stderr)

    remediation_commands = _build_remediation_commands(results, args.keystone_url)
    print("\n--- Remediation Commands ---", file=sys.stderr)
    if remediation_commands:
        for command in remediation_commands:
            print(command, file=sys.stderr)
    else:
        print("none", file=sys.stderr)


if __name__ == "__main__":
    main()
