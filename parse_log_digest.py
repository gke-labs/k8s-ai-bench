#!/usr/bin/env python3
import yaml
import json
import sys
import os
import argparse
from typing import Dict, Any, List

import re

# ANSI Colors
CYAN = "\033[96m"
GREEN = "\033[92m"
YELLOW = "\033[93m"
RED = "\033[91m"
RESET = "\033[0m"
BOLD = "\033[1m"
DIM = "\033[2m"

ANSI_ESCAPE = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')

def strip_ansi(text: str) -> str:
    return ANSI_ESCAPE.sub('', text)

def parse_args():
    parser = argparse.ArgumentParser(description="Parse trace.yaml into a readable digest.")
    parser.add_argument("file", nargs="?", default="trace.yaml", help="Path to trace.yaml file or directory")
    return parser.parse_args()

def out(text: str, file=None):
    if file:
        file.write(strip_ansi(text) + "\n")
    else:
        print(text)

def print_header(role: str, color: str, file=None):
    out(f"\n{color}{BOLD}[{role.upper()}]{RESET}", file=file)

def format_json(data: Any) -> str:
    return json.dumps(data, indent=2)

def process_part(part: Dict[str, Any], role: str, file=None):
    if "text" in part:
        text = part['text'].strip()
        if text:
            out(f"{text}", file=file)
    elif "functionCall" in part:
        fc = part["functionCall"]
        name = fc.get("name", "unknown_tool")
        args = fc.get("args", {})
        out(f"{YELLOW}Tool Call: {name}{RESET}", file=file)
        out(f"{DIM}{format_json(args)}{RESET}", file=file)
    elif "functionResponse" in part:
        fr = part["functionResponse"]
        name = fr.get("name", "unknown_tool")
        response = fr.get("response", {})
        out(f"{YELLOW}Tool Output ({name}):{RESET}", file=file)
        out(f"{GREEN}{format_json(response)}{RESET}", file=file)

def process_content(content: Dict[str, Any], file=None):
    role = content.get("role", "unknown")
    parts = content.get("parts", [])
    
    if role == "model":
        return

    header_color = RESET
    if role == "user":
        header_color = BOLD
    elif role == "function" or role == "tool":
        header_color = GREEN
    
    print_header(role, header_color, file=file)
    for part in parts:
        process_part(part, role, file=file)

def process_event(event_data: Dict[str, Any], file=None):
    candidates = event_data.get("candidates", [])
    for candidate in candidates:
        content = candidate.get("content", {})
        parts = content.get("parts", [])
        role = content.get("role", "model")
        
        print_header(role, CYAN if role == "model" else RESET, file=file)
        for part in parts:
            process_part(part, role, file=file)

def parse_request_body(request_str: str, file=None):
    parts = request_str.split("\r\n\r\n", 1)
    if len(parts) < 2:
        return
    
    body_str = parts[1]
    try:
        data = json.loads(body_str)
        contents = data.get("contents", [])
        if contents:
            last_content = contents[-1]
            process_content(last_content, file=file)
    except json.JSONDecodeError:
        pass

def parse_trace(filepath: str):
    if not os.path.exists(filepath):
        print(f"{RED}Error: File not found: {filepath}{RESET}")
        return

    output_path = os.path.join(os.path.dirname(filepath), "digest.txt")
    print(f"{DIM}Parsing {filepath} -> {output_path}{RESET}")

    try:
        with open(filepath, 'r') as f, open(output_path, 'w') as out_f:
            try:
                documents = yaml.safe_load_all(f)
                
                for doc in documents:
                    if not doc:
                        continue
                    
                    action = doc.get("action")
                    payload = doc.get("payload", {})
                    
                    if action == "http.request":
                        req = payload.get("request", "")
                        parse_request_body(req, file=out_f)
                    
                    elif action == "http.response":
                        body = payload.get("body", "")
                        lines = body.splitlines()
                        for line in lines:
                            if line.startswith("data: "):
                                json_str = line[len("data: "):].strip()
                                try:
                                    data = json.loads(json_str)
                                    process_event(data, file=out_f)
                                except json.JSONDecodeError:
                                    pass
                                    
            except Exception as e:
                print(f"{RED}Error parsing YAML stream: {e}{RESET}")
    except Exception as e:
        print(f"{RED}Error opening files: {e}{RESET}")

def process_path(path: str):
    if os.path.isfile(path):
        parse_trace(path)
    elif os.path.isdir(path):
        trace_files = []
        for root, dirs, files in os.walk(path):
            if "trace.yaml" in files:
                trace_files.append(os.path.join(root, "trace.yaml"))
        
        trace_files.sort()
        
        for trace_file in trace_files:
            parse_trace(trace_file)
    else:
        print(f"{RED}Error: Path not found: {path}{RESET}")

if __name__ == "__main__":
    try:
        from signal import signal, SIGPIPE, SIG_DFL
        signal(SIGPIPE, SIG_DFL)
    except ImportError:
        pass
        
    args = parse_args()
    process_path(args.file)
