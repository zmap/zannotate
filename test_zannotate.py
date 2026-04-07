# test_zannotate.py
import json
import subprocess
import textwrap


def test_rdns():
    tests = [
        ("normal input", "1.1.1.1\n8.8.8.8\n"),
        ("missing trailing new-line", "1.1.1.1\n8.8.8.8"),
    ]
    for test in tests:
        output = run(test[1], "--rdns")
        assert output == unordered(
            [
                {"ip": "1.1.1.1", "rdns": {"domain_names": ["one.one.one.one"]}},
                {"ip": "8.8.8.8", "rdns": {"domain_names": ["dns.google"]}},
            ]
        )


def test_rdns_csv_stdin():
    # Test string input in csv format, will need to extract the ip column
    input_str = textwrap.dedent("""\
    name,ip,date
    cloudflare,1.1.1.1,04-04-26
    google,8.8.8.8,04-04-26
    """)
    print(input_str)
    output = run(
        input_str, "--rdns", "--input-file-type", "csv", "--input-ip-field", "ip"
    )
    assert output == unordered(
        [
            {
                "ip": "1.1.1.1",
                "name": "cloudflare",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "ip": "8.8.8.8",
                "name": "google",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


def test_rdns_csv_stdin_non_standard_key_name():
    # Test string input in csv format, will need to extract the ip column
    input_str = textwrap.dedent("""\
    name,date,ip_address
    cloudflare,04-04-26,1.1.1.1
    google,04-04-26,8.8.8.8
    """)
    output = run(
        input_str,
        "--rdns",
        "--input-file-type",
        "csv",
        "--input-ip-field",
        "ip_address",
    )
    assert output == unordered(
        [
            {
                "ip_address": "1.1.1.1",
                "name": "cloudflare",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "ip_address": "8.8.8.8",
                "name": "google",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


def test_rdns_csv_stdin_non_standard_output_key():
    input_str = textwrap.dedent("""\
    name,date,ip
    cloudflare,04-04-26,1.1.1.1
    google,04-04-26,8.8.8.8
    """)
    output = run(
        input_str,
        "--rdns",
        "--input-file-type",
        "csv",
        "--output-annotation-field",
        "z_annotate_output",
    )
    assert output == unordered(
        [
            {
                "ip": "1.1.1.1",
                "name": "cloudflare",
                "date": "04-04-26",
                "z_annotate_output": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "ip": "8.8.8.8",
                "name": "google",
                "date": "04-04-26",
                "z_annotate_output": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


def test_rdns_csv_file(tmp_path):
    # Test string input in csv format, will need to extract the ip column
    input_str = textwrap.dedent("""\
    name,ip,date
    cloudflare,1.1.1.1,04-04-26
    google,8.8.8.8,04-04-26
    """)
    input_file = tmp_path / "input.csv"
    input_file.write_text(textwrap.dedent("""\
        name,ip,date
        cloudflare,1.1.1.1,04-04-26
        google,8.8.8.8,04-04-26
    """))

    output = run(
        input_str,
        "--rdns",
        "--input-file-type",
        "csv",
        "--input-file",
        str(input_file),
    )
    assert output == unordered(
        [
            {
                "ip": "1.1.1.1",
                "name": "cloudflare",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "ip": "8.8.8.8",
                "name": "google",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


def test_rdns_json_stdin():
    # Test string input in csv format, will need to extract the ip column
    input_str = textwrap.dedent("""\
    {"name": "cloudflare","ip": "1.1.1.1", "date": "04-04-26"}
    {"name": "google","ip": "8.8.8.8", "date": "04-04-26"}
    """)
    output = run(input_str, "--rdns", "--input-file-type", "json")
    assert output == unordered(
        [
            {
                "name": "cloudflare",
                "ip": "1.1.1.1",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "name": "google",
                "ip": "8.8.8.8",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


def test_rdns_json_stdin_non_standard_key_name():
    input_str = textwrap.dedent("""\
    {"name": "cloudflare","ip_address": "1.1.1.1", "date": "04-04-26"}
    {"name": "google","ip_address": "8.8.8.8", "date": "04-04-26"}
    """)
    output = run(
        input_str,
        "--rdns",
        "--input-file-type",
        "json",
        "--input-ip-field",
        "ip_address",
    )
    assert output == unordered(
        [
            {
                "name": "cloudflare",
                "ip_address": "1.1.1.1",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["one.one.one.one"]}},
            },
            {
                "name": "google",
                "ip_address": "8.8.8.8",
                "date": "04-04-26",
                "zannotate": {"rdns": {"domain_names": ["dns.google"]}},
            },
        ]
    )


# Helpers
def run(stdin: str, *args) -> list[dict]:
    result = subprocess.run(
        ["./zannotate", *args],
        input=stdin,
        capture_output=True,
        text=True,
        check=True,
    )
    lines = [json.loads(line) for line in result.stdout.strip().splitlines()]
    return sorted(lines, key=lambda x: json.dumps(x, sort_keys=True))


def unordered(expected: list[dict]):
    """Sort both sides by JSON repr for order-insensitive comparison."""
    return sorted(expected, key=lambda x: json.dumps(x, sort_keys=True))
