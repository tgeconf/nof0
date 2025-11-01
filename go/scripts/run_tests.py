#!/usr/bin/env python3
"""
NOF0 API Test Runner
====================

This script runs both unit tests and integration tests for the NOF0 API project.
It provides a more readable and maintainable alternative to the shell scripts.
The script now supports automatic discovery of test packages.

Usage:
    python scripts/run_tests.py [test_type] [options]

Test types:
    unit          - Run unit tests only (auto-discovers all unit test packages)
    integration   - Run integration tests only (auto-discovers all integration test packages)
    all           - Run all tests (default)
"""

import subprocess
import sys
import os
import signal
import time
import argparse
import threading
import requests
import tempfile
import psutil
from pathlib import Path
from typing import Optional, List

# ANSI color codes for terminal output
COLORS = {
    "GREEN": "\033[0;32m",
    "RED": "\033[0;31m",
    "YELLOW": "\033[1;33m",
    "BLUE": "\033[0;34m",
    "NC": "\033[0m",  # No Color
}


def print_colored(message: str, color: str = "NC"):
    """Print a message with ANSI color codes."""
    print(f"{COLORS[color]}{message}{COLORS['NC']}")


def run_command(
    command: List[str], shell: bool = False, cwd: str = None, timeout: int = None
) -> tuple:
    """
    Run a shell command and return stdout, stderr, and return code.

    Args:
        command: Command to execute as a list of strings
        shell: Whether to run the command through the shell
        cwd: Working directory for the command
        timeout: Optional timeout in seconds

    Returns:
        Tuple of (stdout, stderr, return_code)
    """
    try:
        result = subprocess.run(
            command,
            shell=shell,
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return result.stdout, result.stderr, result.returncode
    except subprocess.TimeoutExpired:
        return "", "Command timed out", 1
    except Exception as e:
        return "", str(e), 1


def check_server_running(port: int = 8888) -> bool:
    """Check if a server is already running on the specified port."""
    try:
        for proc in psutil.process_iter(["pid", "name"]):
            try:
                connections = proc.connections()
                for conn in connections:
                    if conn.laddr.port == port and conn.status == "LISTEN":
                        return True
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue
        return False
    except Exception:
        # Fallback to using netstat if psutil is not available
        stdout, _, _ = run_command(["netstat", "-an"], shell=True)
        return f":{port}" in stdout


def start_server() -> Optional[int]:
    """Start the NOF0 API server and return its PID."""
    print_colored("Building application...", "BLUE")

    # Build the application
    stdout, stderr, code = run_command(["go", "build", "-o", "nof0-api", "./nof0.go"])
    if code != 0:
        print_colored(f"Build failed: {stderr}", "RED")
        return None

    print_colored("✓ Build successful", "GREEN")
    print()

    # Start server
    print_colored("Starting server on port 8888...", "BLUE")

    try:
        with open("server.log", "w") as log_file:
            process = subprocess.Popen(
                ["./nof0-api", "-f", "etc/nof0.yaml"], stdout=log_file, stderr=log_file
            )
            return process.pid
    except Exception as e:
        print_colored(f"Failed to start server: {e}", "RED")
        return None


def wait_for_server(timeout: int = 30) -> bool:
    """Wait for the server to be ready to accept requests."""
    print("Waiting for server to be ready...", end="")

    for i in range(timeout):
        try:
            response = requests.get(
                "http://localhost:8888/api/crypto-prices", timeout=5
            )
            if response.status_code == 200:
                print_colored(" ✓ Server ready", "GREEN")
                return True
        except requests.exceptions.RequestException:
            pass

        time.sleep(1)
        print(".", end="", flush=True)

    print_colored(" ✗ Server failed to start within timeout", "RED")
    return False


def stop_server(pid: int):
    """Stop the server process."""
    if pid:
        print_colored("Stopping server...", "BLUE")
        try:
            process = psutil.Process(pid)
            process.terminate()
            process.wait(timeout=5)
            print_colored("✓ Server stopped", "GREEN")
        except psutil.NoSuchProcess:
            pass
        except psutil.TimeoutExpired:
            try:
                process.kill()
                print_colored("✓ Server killed", "GREEN")
            except psutil.NoSuchProcess:
                pass
        except Exception as e:
            print_colored(f"Error stopping server: {e}", "RED")


def discover_test_packages() -> List[str]:
    """Discover all Go packages that contain unit tests."""
    print_colored("Discovering test packages...", "BLUE")
    print()

    # Common directories that might contain unit tests
    potential_dirs = [
        "./internal/...",
        "./pkg/llm",
        "./pkg/market",
        "./pkg/market/exchanges/hyperliquid",
        "./pkg/market/indicators",
        "./pkg/executor",
        "./pkg/manager",
        "./pkg/backtest",
        "./pkg/exchange",
        "./pkg/prompt",
    ]

    test_packages = []

    for dir_pattern in potential_dirs:
        # Remove ... suffix for directory check
        dir_path = dir_pattern.replace("/...", "")
        if not os.path.exists(dir_path):
            continue

        # Look for test files in the directory
        test_files = []
        try:
            for root, dirs, files in os.walk(dir_path):
                for file in files:
                    if file.endswith("_test.go") and not file.endswith(
                        "_integration_test.go"
                    ):
                        test_files.append(os.path.join(root, file))
        except Exception as e:
            print_colored(f"Error scanning {dir_path}: {e}", "YELLOW")
            continue

        if test_files:
            # Use Go package path
            package_path = dir_pattern
            print_colored(
                f"✓ Found unit tests in {package_path}: {len(test_files)} files",
                "GREEN",
            )
            test_packages.append(dir_pattern)
        else:
            print_colored(
                f"✗ No unit tests found in {dir_path.replace('./', '')}", "RED"
            )

    print()
    return test_packages


def run_unit_tests() -> bool:
    """Run unit tests and return success status."""
    print_colored("Running unit tests with auto-discovery...", "YELLOW")
    print()

    success = True

    # Discover test packages
    test_packages = discover_test_packages()

    if not test_packages:
        print_colored("No test packages found", "YELLOW")
        return True

    # Run tests for discovered packages
    for package in test_packages:
        print_colored(f"Running unit tests for {package}...", "BLUE")

        stdout, stderr, code = run_command(
            ["go", "test", "-tags=dotenv", package, "-v"]
        )

        if code == 0:
            print_colored(f"✓ {package} unit tests passed", "GREEN")
        else:
            print_colored(f"✗ {package} unit tests failed", "RED")
            success = False

        if stdout:
            print(stdout)
        if stderr:
            print(stderr, file=sys.stderr)

        print()

    return success


def run_benchmarks():
    """Run benchmarks and display results."""
    print_colored("Running benchmarks...", "YELLOW")
    print()

    stdout, stderr, _ = run_command(
        [
            "go",
            "test",
            "-tags=dotenv",
            "./internal/...",
            "-bench=.",
            "-benchmem",
            "-run=^$",
        ]
    )

    # Filter to show only benchmark results
    if stdout:
        lines = stdout.split("\n")
        for line in lines:
            if "Benchmark" in line or "PASS" in line:
                print(line)

    if stderr:
        print(stderr, file=sys.stderr)


def generate_coverage():
    """Generate test coverage report."""
    print_colored("Generating coverage report...", "YELLOW")
    print()

    # Generate coverage file
    stdout, stderr, _ = run_command(
        ["go", "test", "-tags=dotenv", "./internal/...", "-coverprofile=coverage.out"]
    )

    # Extract coverage percentage
    if os.path.exists("coverage.out"):
        stdout, stderr, _ = run_command(["go", "tool", "cover", "-func=coverage.out"])
        if stdout:
            for line in stdout.split("\n"):
                if "total:" in line:
                    coverage = line.split()[-1]
                    print_colored(f"Total Coverage: {coverage}", "GREEN")
                    break

    if stderr:
        print(stderr, file=sys.stderr)

    print()
    print("Coverage report: coverage.out")
    print("View HTML report: go tool cover -html=coverage.out")


def discover_integration_test_packages() -> List[str]:
    """Discover all Go packages that contain integration tests."""
    print_colored("Discovering integration test packages...", "BLUE")
    print()

    # Common directories that might contain integration tests
    potential_dirs = [
        "./pkg/llm",
        "./pkg/market",
        "./pkg/market/exchanges/hyperliquid",
        "./pkg/market/indicators",
        "./internal/config",
        "./test",
    ]

    integration_packages = []

    for dir_path in potential_dirs:
        # Check if directory exists
        if not os.path.exists(dir_path):
            continue

        # Look for integration test files in the directory
        integration_test_files = []
        try:
            for root, dirs, files in os.walk(dir_path):
                for file in files:
                    if file.endswith("_integration_test.go"):
                        integration_test_files.append(os.path.join(root, file))
        except Exception as e:
            print_colored(f"Error scanning {dir_path}: {e}", "YELLOW")
            continue

        if integration_test_files:
            # Use Go package path (replace slashes with dots for display)
            package_path = dir_path.replace("./", "").replace("/", ".")
            print_colored(
                f"✓ Found integration tests in {package_path}: {len(integration_test_files)} files",
                "GREEN",
            )
            integration_packages.append(dir_path)
        else:
            print_colored(
                f"✗ No integration tests found in {dir_path.replace('./', '')}", "RED"
            )

    print()
    return integration_packages


def run_integration_tests() -> bool:
    """Run integration tests and return success status."""
    print_colored("Running integration tests with auto-discovery...", "YELLOW")
    print()

    success = True

    # Discover integration test packages
    integration_packages = discover_integration_test_packages()

    if not integration_packages:
        print_colored("No integration test packages found", "YELLOW")
        return True

    # Run integration tests for discovered packages
    for package in integration_packages:
        print_colored(f"Running integration tests for {package}...", "BLUE")

        stdout, stderr, code = run_command(
            [
                "go",
                "test",
                "-tags=dotenv integration",
                package,
                "-v",
                "-run",
                "Integration",
            ]
        )

        if code == 0:
            print_colored(f"✓ {package} integration tests passed", "GREEN")
        else:
            print_colored(f"✗ {package} integration tests failed", "RED")
            success = False

        if stdout:
            print(stdout)
        if stderr:
            print(stderr, file=sys.stderr)

        print()

    # Run API integration tests separately (they might have different tag requirements)
    print_colored("Running API integration tests...", "BLUE")
    stdout, stderr, code = run_command(
        ["go", "test", "-tags=dotenv", "./test/...", "-v"]
    )

    if code == 0:
        print_colored("✓ API integration tests passed", "GREEN")
    else:
        print_colored("✗ API integration tests failed", "RED")
        success = False

    if stdout:
        print(stdout)
    if stderr:
        print(stderr, file=sys.stderr)

    print()
    return success


def test_endpoints() -> bool:
    """Test individual API endpoints."""
    print_colored("Testing individual endpoints...", "YELLOW")
    print()

    endpoints = [
        ("crypto-prices", "Crypto Prices"),
        ("leaderboard", "Leaderboard"),
        ("trades", "Trades"),
        ("since-inception-values", "Since Inception"),
        ("account-totals", "Account Totals"),
        ("analytics", "Analytics"),
        ("analytics/qwen3-max", "Model Analytics"),
    ]

    all_passed = True

    for endpoint, name in endpoints:
        print(f"Testing {name}... ", end="", flush=True)

        try:
            response = requests.get(f"http://localhost:8888/api/{endpoint}", timeout=5)
            if response.status_code == 200:
                print_colored("✓", "GREEN")
            else:
                print_colored("✗", "RED")
                all_passed = False
        except requests.exceptions.RequestException:
            print_colored("✗", "RED")
            all_passed = False

    return all_passed


def show_server_log():
    """Display the server log if it exists."""
    if os.path.exists("server.log"):
        print("Server log:")
        with open("server.log", "r") as f:
            print(f.read())


def run_all_tests() -> int:
    """Run all tests and return exit code."""
    print_colored("================================", "NC")
    print_colored("NOF0 API Test Suite", "NC")
    print_colored("================================", "NC")
    print()

    exit_code = 0

    # Run unit tests
    if not run_unit_tests():
        exit_code = 1

    # Run benchmarks
    run_benchmarks()

    # Generate coverage
    generate_coverage()

    print()
    print_colored("=========================================", "NC")
    print_colored("NOF0 API Integration Test Suite", "NC")
    print_colored("=========================================", "NC")
    print()

    server_pid = None
    server_started = False

    # Check if server is already running
    if check_server_running(8888):
        print_colored("⚠ Server already running on port 8888", "YELLOW")
        print("Using existing server instance...")
    else:
        server_pid = start_server()
        if server_pid:
            server_started = True
            if not wait_for_server():
                stop_server(server_pid)
                return 1

    try:
        integration_success = run_integration_tests()
        endpoints_success = test_endpoints()

        if not (integration_success and endpoints_success):
            exit_code = 1

    finally:
        # Cleanup
        if server_started and server_pid:
            stop_server(server_pid)
            if exit_code != 0:
                show_server_log()

    print()
    print_colored("=========================================", "NC")
    if exit_code == 0:
        print_colored("All tests completed successfully!", "GREEN")
    else:
        print_colored("Some tests failed!", "RED")
    print_colored("=========================================", "NC")
    print()

    return exit_code


def main():
    """Main entry point for the test runner."""
    parser = argparse.ArgumentParser(description="NOF0 API Test Runner")
    parser.add_argument(
        "test_type",
        choices=["unit", "integration", "all"],
        default="all",
        nargs="?",
        help="Type of tests to run (default: all)",
    )
    parser.add_argument(
        "--no-server",
        action="store_true",
        help="Skip server startup for integration tests",
    )

    args = parser.parse_args()

    # Change to project root directory
    script_dir = Path(__file__).parent
    os.chdir(script_dir / "..")

    try:
        if args.test_type == "unit":
            print_colored("Running unit tests only...", "NC")
            success = run_unit_tests()
            run_benchmarks()
            generate_coverage()
            return 0 if success else 1

        elif args.test_type == "integration":
            print_colored("Running integration tests only...", "NC")

            server_pid = None
            if not args.no_server:
                if not check_server_running(8888):
                    server_pid = start_server()
                    if server_pid:
                        if not wait_for_server():
                            return 1

            try:
                success = run_integration_tests() and test_endpoints()
                return 0 if success else 1
            finally:
                if server_pid:
                    stop_server(server_pid)

        else:  # 'all'
            return run_all_tests()

    except KeyboardInterrupt:
        print_colored("\nTest execution interrupted", "YELLOW")
        return 1


if __name__ == "__main__":
    sys.exit(main())
