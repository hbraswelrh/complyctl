#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""This script checks for and adds an SPDX line in the comments of source files."""
from __future__ import annotations

import argparse
from collections.abc import Sequence
from pathlib import Path


SPDX_LINE = "SPDX-License-Identifier: Apache-2.0"
COMMENT_STYLES = {
    ".py": "#",
    ".sh": "#",
    ".go": "//",
}


def _load_file(file_path: Path) -> list[str]:
    """Reads a file and returns its lines as a list."""
    try:
        with file_path.open("r", encoding="utf-8") as f:
            return f.readlines()
    except Exception as e:
        print(f"Error loading file {file_path}: {e}")
        return []


def _check_spdx(lines: list[str]) -> bool:
    """Checks if the SPDX line is present in the first lines of a file."""
    for line in lines[:3]:
        if SPDX_LINE in line:
            return True


def _write_file(file_path: Path, lines: list[str]) -> None:
    """Writes modified lines back to the file."""
    try:
        with file_path.open("w", encoding="utf-8") as f:
            f.writelines(lines)
    except Exception as e:
        print(f"Error writing to file {file_path}: {e}")


def _fix_spdx(file_path: Path, lines: list[str]) -> None:
    """Adds an SPDX-License-Identifier line if missing."""
    comment_prefix = COMMENT_STYLES.get(file_path.suffix)
    if not comment_prefix:
        print(f"Skipping {file_path} (unknown file type)")
        return

    spdx_header = f"{comment_prefix} {SPDX_LINE}\n"

    if not lines:
        line_index = 0
    elif lines[0].startswith("#!"):
        line_index = 1
    else:
        line_index = 0

    lines.insert(line_index, spdx_header)
    _write_file(file_path, lines)
    print(f"Fixed SPDX line in {file_path}")


def check_spdx(file_paths: list[str]) -> int:
    """Checks and fixes SPDX license headers in given files."""
    any_fixed = False

    for file_path in file_paths:
        path = Path(file_path)
        lines = _load_file(path)
        if not _check_spdx(lines):
            _fix_spdx(path, lines)
            any_fixed = True
    return 1 if any_fixed else 0


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("filenames", nargs="*")
    args = parser.parse_args(argv)
    return check_spdx(args.filenames)


if __name__ == "__main__":
    raise SystemExit(main())
