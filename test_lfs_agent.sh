#!/bin/bash

# Simulate an init event from Git LFS
echo '{
    "event": "init",
    "operation": "upload",
    "remote": "origin",
    "concurrent": true,
    "concurrenttransfers": 8,
    "objects": [
        {
            "oid": "1234567890abcdef",
            "size": 123456
        },
        {
            "oid": "fedcba0987654321",
            "size": 654321
        }
    ]
}' | /usr/local/bin/lfs-transfer-agent

# Simulate a complete event for upload
echo '{
    "event": "complete",
    "operation": "upload",
    "objects": [
        {
            "oid": "1234567890abcdef",
            "size": 123456
        },
        {
            "oid": "fedcba0987654321",
            "size": 654321
        }
    ]
}' | /usr/local/bin/lfs-transfer-agent
