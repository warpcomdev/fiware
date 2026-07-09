#!/usr/bin/env python3

import argparse
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path


DEFAULT_DOMAINS_DIR = Path(__file__).parent / "domains"
DEFAULT_SUMMARY_PATH = Path(__file__).parent / "requery_empty_userroles_failures.json"


class FiwareQueryError(RuntimeError):
    def __init__(self, message: str, details: dict | None = None):
        super().__init__(message)
        self.details = details or {}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Re-query empty user role dumps using the fiware CLI."
    )
    parser.add_argument(
        "--fiware-bin",
        default="fiware",
        help="Path to the fiware CLI binary. Default: %(default)s",
    )
    parser.add_argument(
        "--domains-dir",
        default=str(DEFAULT_DOMAINS_DIR),
        help="Directory containing dumped domain data. Default: %(default)s",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the users that would be queried without rewriting files.",
    )
    parser.add_argument(
        "--failure-summary",
        default=str(DEFAULT_SUMMARY_PATH),
        help="Path to write the JSON summary of failed re-queries. Default: %(default)s",
    )
    parser.add_argument(
        "--normalize-failure-summary",
        help=(
            "Rewrite an existing failure summary by reparsing each error and "
            "filling structured missing-resource fields."
        ),
    )
    return parser.parse_args()


def load_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as fh:
        return json.load(fh)


def write_json_atomic(path: Path, data: dict) -> None:
    with tempfile.NamedTemporaryFile(
        "w", encoding="utf-8", dir=path.parent, delete=False
    ) as tmp:
        json.dump(data, tmp, indent=2, sort_keys=False)
        tmp.write("\n")
        tmp_path = Path(tmp.name)
    tmp_path.replace(path)


def is_empty_role_dump(path: Path) -> bool:
    try:
        data = load_json(path)
    except json.JSONDecodeError:
        return False
    return data == {}


def extract_error_details(message: str) -> dict:
    details: dict[str, str] = {}

    missing_match = re.search(
        r"Could not find\s+([A-Za-z_ ]+):\s*([A-Fa-f0-9]+)\.",
        message,
    )
    if missing_match:
        resource_type = missing_match.group(1).strip().lower().replace(" ", "_")
        resource_id = missing_match.group(2)
        details["missing_resource_type"] = resource_type
        details["missing_resource_id"] = resource_id
        details[f"missing_{resource_type}_id"] = resource_id

    code_match = re.search(r"failed with code\s+(\d+)", message)
    if code_match:
        details["http_status"] = code_match.group(1)

    return details


def run_fiware(fiware_bin: str, domain_name: str, user_id: str) -> dict:
    cmd = [fiware_bin, "get", "--domain", domain_name, "--userid", user_id, "userroles"]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        stderr = proc.stderr.strip()
        stdout = proc.stdout.strip()
        message = stderr or stdout or f"command failed with exit code {proc.returncode}"
        raise FiwareQueryError(message, extract_error_details(message))
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise FiwareQueryError(f"invalid JSON output: {exc}") from exc


def user_name_from_rolemap(rolemap: dict, user_id: str) -> str:
    for user in rolemap.get("users", []):
        if user.get("id") == user_id:
            return user.get("name") or user_id
    return user_id


def iter_empty_userrole_files(domains_dir: Path) -> list[tuple[str, Path, str]]:
    empty_files: list[tuple[str, Path, str]] = []
    for domain_dir in sorted(domains_dir.iterdir()):
        if not domain_dir.is_dir():
            continue
        for userroles_path in sorted(domain_dir.glob("userroles_*.json")):
            if not is_empty_role_dump(userroles_path):
                continue
            user_id = userroles_path.stem.removeprefix("userroles_")
            empty_files.append((domain_dir.name, userroles_path, user_id))
    return empty_files


def build_failure_entry(
    domain_name: str,
    user_name: str,
    user_id: str,
    exc: Exception,
) -> dict:
    details = exc.details if isinstance(exc, FiwareQueryError) else {}
    return {
        "service": domain_name,
        "user_name": user_name,
        "user_id": user_id,
        "missing_project_id": details.get("missing_project_id"),
        "missing_domain_id": details.get("missing_domain_id"),
        "missing_resource_type": details.get("missing_resource_type"),
        "missing_resource_id": details.get("missing_resource_id"),
        "http_status": details.get("http_status"),
        "error": str(exc),
    }


def enrich_failure_entry(service: str, entry: dict) -> dict:
    details = extract_error_details(entry.get("error", ""))
    return {
        "service": service,
        "user_name": entry.get("user_name"),
        "user_id": entry.get("user_id"),
        "missing_project_id": details.get(
            "missing_project_id", entry.get("missing_project_id")
        ),
        "missing_domain_id": details.get(
            "missing_domain_id", entry.get("missing_domain_id")
        ),
        "missing_resource_type": details.get(
            "missing_resource_type", entry.get("missing_resource_type")
        ),
        "missing_resource_id": details.get(
            "missing_resource_id", entry.get("missing_resource_id")
        ),
        "http_status": details.get("http_status", entry.get("http_status")),
        "error": entry.get("error"),
    }


def normalize_failure_summary(path: Path) -> int:
    data = load_json(path)
    services = data.get("services", {})
    normalized_services: dict[str, list[dict]] = {}

    for service, entries in services.items():
        normalized_services[service] = [
            enrich_failure_entry(service, entry) for entry in entries
        ]

    normalized = {
        "processed": data.get("processed"),
        "updated": data.get("updated"),
        "failures": data.get("failures"),
        "services": normalized_services,
    }
    write_json_atomic(path, normalized)
    print(f"normalized failure summary at {path}", file=sys.stderr)
    return 0


def main() -> int:
    args = parse_args()
    if args.normalize_failure_summary:
        return normalize_failure_summary(Path(args.normalize_failure_summary))

    domains_dir = Path(args.domains_dir)
    if not domains_dir.is_dir():
        print(f"domains dir not found: {domains_dir}", file=sys.stderr)
        return 1

    empty_files = iter_empty_userrole_files(domains_dir)
    if not empty_files:
        print("no empty user role dumps found", file=sys.stderr)
        return 0

    failures = 0
    updated = 0
    failure_entries: list[dict] = []

    for domain_name, userroles_path, user_id in empty_files:
        rolemap_path = userroles_path.parent / "rolemap.json"
        rolemap = load_json(rolemap_path) if rolemap_path.exists() else {}
        user_name = user_name_from_rolemap(rolemap, user_id)
        print(
            f"[requery] {domain_name}: {user_name} (user ID: {user_id})",
            file=sys.stderr,
        )

        if args.dry_run:
            continue

        try:
            data = run_fiware(args.fiware_bin, domain_name, user_id)
            write_json_atomic(userroles_path, data)
            updated += 1
        except Exception as exc:
            failures += 1
            failure_entries.append(
                build_failure_entry(domain_name, user_name, user_id, exc)
            )
            print(
                f"failed to re-query {domain_name}/{user_id}: {exc}",
                file=sys.stderr,
            )

    if not args.dry_run:
        summary = {
            "processed": len(empty_files),
            "updated": updated,
            "failures": failures,
            "services": {},
        }
        for entry in failure_entries:
            service = entry["service"]
            service_bucket = summary["services"].setdefault(service, [])
            service_bucket.append(
                {
                    "user_name": entry["user_name"],
                    "user_id": entry["user_id"],
                    "missing_project_id": entry["missing_project_id"],
                    "missing_domain_id": entry["missing_domain_id"],
                    "missing_resource_type": entry["missing_resource_type"],
                    "missing_resource_id": entry["missing_resource_id"],
                    "http_status": entry["http_status"],
                    "error": entry["error"],
                }
            )
        write_json_atomic(Path(args.failure_summary), summary)
        print(f"failure summary written to {args.failure_summary}", file=sys.stderr)

    print(
        f"processed={len(empty_files)} updated={updated} failures={failures}",
        file=sys.stderr,
    )
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
