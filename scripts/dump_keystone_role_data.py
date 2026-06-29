#!/usr/bin/env python3

import argparse
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Dump FIWARE Keystone role data per domain using the fiware CLI."
    )
    parser.add_argument(
        "--fiware-bin",
        default="fiware",
        help="Path to the fiware CLI binary. Default: %(default)s",
    )
    parser.add_argument(
        "--outdir",
        default="domains",
        help="Directory where domain data will be stored. Default: %(default)s",
    )
    parser.add_argument(
        "--include-admin-domain",
        action="store_true",
        help="Include admin_domain in the dump.",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Redownload files even if they already exist.",
    )
    parser.add_argument(
        "--continue-on-error",
        action="store_true",
        help="Continue processing other domains if one command fails.",
    )
    return parser.parse_args()


def safe_domain_dirname(domain_name: str) -> str:
    return re.sub(r"[^A-Za-z0-9._-]", "_", domain_name)


def run_fiware(fiware_bin: str, args: list[str]) -> dict:
    cmd = [fiware_bin, "get", *args]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        stderr = proc.stderr.strip()
        stdout = proc.stdout.strip()
        message = stderr or stdout or f"command failed with exit code {proc.returncode}"
        raise RuntimeError(f"{' '.join(cmd)}: {message}")
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"{' '.join(cmd)}: invalid JSON output: {exc}") from exc


def write_json_atomic(path: Path, data: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w", encoding="utf-8", dir=path.parent, delete=False
    ) as tmp:
        json.dump(data, tmp, indent=2, sort_keys=False)
        tmp.write("\n")
        tmp_path = Path(tmp.name)
    tmp_path.replace(path)


def dump_if_missing(
    target: Path,
    fiware_bin: str,
    fiware_args: list[str],
    force: bool,
) -> dict:
    if target.exists() and not force:
        with target.open(encoding="utf-8") as fh:
            return json.load(fh)
    data = run_fiware(fiware_bin, fiware_args)
    write_json_atomic(target, data)
    return data


def iter_domains(domains_payload: dict, include_admin_domain: bool) -> list[dict]:
    domains = domains_payload.get("domains", [])
    if include_admin_domain:
        return domains
    return [domain for domain in domains if domain.get("name") != "admin_domain"]


def dump_domain(
    fiware_bin: str,
    base_dir: Path,
    domain_name: str,
    force: bool,
) -> None:
    domain_dir = base_dir / safe_domain_dirname(domain_name)
    print(f"[domain] {domain_name}", file=sys.stderr)

    rolemap_path = domain_dir / "rolemap.json"
    rolemap = dump_if_missing(
        rolemap_path,
        fiware_bin,
        ["--domain", domain_name, "rolemap"],
        force,
    )

    for user in rolemap.get("users", []):
        user_id = user.get("id")
        if not user_id:
            continue
        userroles_path = domain_dir / f"userroles_{user_id}.json"
        try:
            dump_if_missing(
                userroles_path,
                fiware_bin,
                ["--domain", domain_name, "--userid", user_id, "userroles"],
                force,
            )
        except:
            print(f"Failed to collect data for user {user_id}")
            with open(userroles_path, "w+", encoding="utf-8") as outf:
                outf.write("{}")

    for group in rolemap.get("groups", []):
        group_id = group.get("id")
        if not group_id:
            continue
        grouproles_path = domain_dir / f"grouproles_{group_id}.json"
        dump_if_missing(
            grouproles_path,
            fiware_bin,
            ["--domain", domain_name, "--groupid", group_id, "grouproles"],
            force,
        )


def main() -> int:
    args = parse_args()
    outdir = Path(args.outdir)
    outdir.mkdir(parents=True, exist_ok=True)

    domains_path = outdir / "_domains.json"
    try:
        domains_payload = dump_if_missing(
            domains_path,
            args.fiware_bin,
            ["domains"],
            args.force,
        )
    except Exception as exc:
        print(f"failed to dump domains: {exc}", file=sys.stderr)
        return 1

    failures = 0
    for domain in iter_domains(domains_payload, args.include_admin_domain):
        domain_name = domain.get("name")
        if not domain_name:
            continue
        try:
            dump_domain(args.fiware_bin, outdir, domain_name, args.force)
        except Exception as exc:
            failures += 1
            print(f"failed domain {domain_name}: {exc}", file=sys.stderr)
            if not args.continue_on_error:
                return 1

    if failures:
        print(f"completed with {failures} domain failures", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
