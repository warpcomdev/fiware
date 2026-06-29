#!/usr/bin/env python3

import argparse
import json
import sys
from pathlib import Path


DOMAINS_DIR = Path(__file__).parent / "domains"

ADMIN_SPECIFIC = {"ServiceAdmin" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")}
CUSTOMER_SPECIFIC = {"ServiceCustomer" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")}
SUB_ADMIN_SPECIFIC = {"SubServiceAdmin" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")}
SUB_CUSTOMER_SPECIFIC = {"SubServiceCustomer" + s for s in ("IOTAGENT", "ORION", "PERSEO", "STH")}

GENERIC_ADMIN = {"admin", "ServiceAdmin"}
GENERIC_CUSTOMER = {"ServiceCustomer"}
GENERIC_SUB_ADMIN = {"SubServiceAdmin"}
GENERIC_SUB_CUSTOMER = {"SubServiceCustomer"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Summarize Keystone role assignments per domain."
    )
    parser.add_argument(
        "-v", "--verbose", action="store_true", help="Print all mappings without filtering."
    )
    return parser.parse_args()


def load_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def process_domain(domain_dir: Path) -> tuple[dict, dict]:
    rolemap_path = domain_dir / "rolemap.json"
    if not rolemap_path.exists():
        return {}, {}

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

    user_roles: dict[tuple[str, str], list[str]] = {}
    group_scopes: dict[tuple[str, str], list[str]] = {}

    for fpath in sorted(domain_dir.glob("userroles_*.json")):
        data = load_json(fpath)
        for assign in data.get("assignments", []):
            user_info = assign.get("user", {})
            uname = user_info.get("name")
            if not uname:
                uid = user_info.get("id", "")
                uname = user_id_to_name.get(uid, uid)

            scope_name = assign.get("scope_name", "")
            role_name = assign.get("role", {}).get("name", "")
            if not scope_name or not role_name:
                continue

            key = (uname, scope_name)
            if key not in user_roles:
                user_roles[key] = []
            if role_name not in user_roles[key]:
                user_roles[key].append(role_name)

    for fpath in sorted(domain_dir.glob("grouproles_*.json")):
        data = load_json(fpath)
        for assign in data.get("assignments", []):
            group_info = assign.get("group", {})
            gname = group_info.get("name")
            if not gname:
                gid = group_info.get("id", "")
                gname = group_id_to_name.get(gid, gid)

            scope_name = assign.get("scope_name", "")
            role_name = assign.get("role", {}).get("name", "")
            if not scope_name or not role_name:
                continue

            key = (gname, role_name)
            if key not in group_scopes:
                group_scopes[key] = []
            if scope_name not in group_scopes[key]:
                group_scopes[key].append(scope_name)

    return user_roles, group_scopes


def _matches_filter(roles: list[str]) -> bool:
    role_set = set(roles)
    if (role_set & ADMIN_SPECIFIC) and not (role_set & GENERIC_ADMIN):
        return True
    if (role_set & CUSTOMER_SPECIFIC) and not (role_set & GENERIC_CUSTOMER):
        return True
    if (role_set & SUB_ADMIN_SPECIFIC) and not (role_set & GENERIC_SUB_ADMIN):
        return True
    if (role_set & SUB_CUSTOMER_SPECIFIC) and not (role_set & GENERIC_SUB_CUSTOMER):
        return True
    return False


def _tuple_key_dict(d: dict, verbose: bool) -> dict:
    result = {}
    for k, v in d.items():
        if verbose or _matches_filter(v):
            result[f"{k[0]}|{k[1]}"] = v
    return result


def main() -> None:
    args = parse_args()
    results: dict[str, dict] = {}

    for domain_dir in sorted(DOMAINS_DIR.iterdir()):
        if not domain_dir.is_dir():
            continue

        user_roles, group_scopes = process_domain(domain_dir)
        filtered_user = _tuple_key_dict(user_roles, args.verbose)
        filtered_group = _tuple_key_dict(group_scopes, args.verbose)
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
        print(f"[{domain_name}] {len(users)} users, {len(groups)} groups", file=sys.stderr)
        for u in users:
            print(f"  user: {u}", file=sys.stderr)
        for g in groups:
            print(f"  group: {g}", file=sys.stderr)


if __name__ == "__main__":
    main()
