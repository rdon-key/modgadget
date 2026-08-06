#!/usr/bin/env python3
"""Build the maintained Cardputer ADV examples with TinyGo."""

from __future__ import annotations

import argparse
import subprocess
import tempfile
from pathlib import Path


EXAMPLES = (
    "hello-text",
    "multilingual-text",
    "text-ticker",
    "keyboard-typing",
    "cardputer-adv-audio",
    "rdon-type100",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tinygo", default="tinygo", help="TinyGo executable (default: tinygo)")
    parser.add_argument(
        "--target",
        default="m5stamp-s3a",
        help="Cardputer ADV TinyGo target name or JSON path (default: m5stamp-s3a)",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    repository = Path(__file__).resolve().parents[1]
    with tempfile.TemporaryDirectory(prefix="modgadget-examples-") as temporary:
        output_directory = Path(temporary)
        for example in EXAMPLES:
            print(f"building {example}", flush=True)
            command = [
                args.tinygo,
                "build",
                f"-target={args.target}",
                "-o",
                str(output_directory / f"{example}.bin"),
                f"./examples/{example}",
            ]
            subprocess.run(command, cwd=repository, check=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
