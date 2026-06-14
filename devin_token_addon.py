#!/usr/bin/env python3
"""
Mitmproxy addon: Capture live Devin CLI data (tokens + quota only).
Model configs and session info are read directly by the Go harness from disk.
Writes to /tmp/devin_live.json.
"""

import struct
import json
from mitmproxy import http

STATE_FILE = '/tmp/devin_live.json'

def read_varint(data, pos):
    value = 0; shift = 0; br = 0
    while pos + br < len(data):
        b = data[pos + br]; br += 1
        value |= (b & 0x7F) << shift
        if not (b & 0x80): return value, br
        shift += 7
        if shift >= 64: break
    return 0, 0

def parse_connect_stream(data):
    messages = []
    pos = 0
    while pos + 5 <= len(data):
        flags = data[pos]
        length = struct.unpack('>I', data[pos+1:pos+5])[0]
        pos += 5
        if length > 10_000_000: break
        if pos + length <= len(data):
            messages.append(data[pos:pos+length])
            pos += length
        else: break
    return messages

def extract_tokens(msg_data):
    it = ot = None
    pos = 0
    while pos < len(msg_data):
        tag, vb = read_varint(msg_data, pos)
        if vb == 0: break
        pos += vb
        fn = tag >> 3; wt = tag & 0x7
        if wt == 2:
            length, lb = read_varint(msg_data, pos); pos += lb
            if pos + length <= len(msg_data):
                sub = msg_data[pos:pos+length]; pos += length
                if fn == 7:
                    sp = 0
                    while sp < len(sub):
                        stag, svb = read_varint(sub, sp)
                        if svb == 0: break
                        sp += svb
                        sf = stag >> 3; sw = stag & 0x7
                        if sw == 0:
                            val, vv = read_varint(sub, sp); sp += vv
                            if sf == 2: it = val
                            elif sf == 3: ot = val
                        elif sw == 2:
                            slen, slb2 = read_varint(sub, sp); sp += slb2 + slen
                        else: sp += 4 if sw == 5 else 8
            else: break
        elif wt == 0: _, vv = read_varint(msg_data, pos); pos += vv
        elif wt == 1: pos += 8
        elif wt == 5: pos += 4
        else: break
    return it, ot

def extract_model(messages):
    if not messages: return None
    msg = messages[0]
    pos = 0
    while pos < len(msg):
        tag, vb = read_varint(msg, pos)
        if vb == 0: break
        pos += vb
        fn = tag >> 3; wt = tag & 0x7
        if wt == 2:
            length, lb = read_varint(msg, pos); pos += lb
            if fn == 7 and pos + length <= len(msg):
                sub = msg[pos:pos+length]; sp = 0
                while sp < len(sub):
                    stag, svb = read_varint(sub, sp)
                    if svb == 0: break
                    sp += svb
                    sfn = stag >> 3; swt = stag & 0x7
                    if swt == 2:
                        slen, slb2 = read_varint(sub, sp); sp += slb2
                        if sfn == 9: return sub[sp:sp+slen].decode('utf-8', errors='replace')
                        sp += slen
                    else: sp += 1 if swt == 0 else (4 if swt == 5 else 8)
            pos += length
        elif wt == 0: _, vv = read_varint(msg, pos); pos += vv
        elif wt == 1: pos += 8
        elif wt == 5: pos += 4
        else: break
    return None

def extract_quota(data):
    result = {}
    pos = 0
    while pos < len(data):
        tag, vb = read_varint(data, pos)
        if vb == 0: break
        pos += vb
        fn = tag >> 3; wt = tag & 0x7
        if wt == 2:
            length, lb = read_varint(data, pos); pos += lb
            if pos + length <= len(data):
                sub = data[pos:pos+length]; pos += length
                if fn == 2:
                    sp = 0
                    while sp < len(sub):
                        stag, svb = read_varint(sub, sp)
                        if svb == 0: break
                        sp += svb
                        sfn = stag >> 3; swt = stag & 0x7
                        if swt == 0:
                            val, vv = read_varint(sub, sp); sp += vv
                            if sfn == 7: result['daily_limit'] = val
                            elif sfn == 8: result['daily_used'] = val
                        elif swt == 2:
                            slen, slb2 = read_varint(sub, sp); sp += slb2
                            if sfn == 2:
                                result['plan'] = sub[sp:sp+slen].decode('utf-8', errors='replace')
                            sp += slen
                        else: sp += 4 if swt == 5 else 8
        elif wt == 0: _, vv = read_varint(data, pos); pos += vv
        else: pos += 4 if swt == 5 else 8
    return result

def response(flow: http.HTTPFlow) -> None:
    path = flow.request.path
    try:
        state = {}
        if os.path.exists(STATE_FILE):
            try:
                with open(STATE_FILE) as f: state = json.load(f)
            except: pass

        if 'GetChatMessage' in path:
            data = flow.response.content
            if data:
                msgs = parse_connect_stream(data)
                if len(msgs) >= 2:
                    model = extract_model(msgs) or state.get('model', 'unknown')
                    for msg in reversed(msgs[-5:]):
                        it, ot = extract_tokens(msg)
                        if it is not None and ot is not None:
                            state['input_tokens'] = it
                            state['output_tokens'] = ot
                            state['model'] = model
                            with open(STATE_FILE, 'w') as f: json.dump(state, f)
                            print(f"[devin-live] {model}: {it} in / {ot} out")
                            return

        elif 'GetUserStatus' in path:
            data = flow.response.content
            if data:
                quota = extract_quota(data)
                if quota:
                    state['quota'] = quota
                    with open(STATE_FILE, 'w') as f: json.dump(state, f)
                    print(f"[devin-live] quota: {quota}")
    except Exception as e:
        print(f"[devin-live] error: {e}")

import os
addons = []