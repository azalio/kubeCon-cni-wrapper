#!/bin/bash
#
# Tenant Routing Setup Script
#
# Purpose: Sets up policy routing for tenant-aware egress
#
# This script creates:
#   - Routing tables for each tenant
#   - IP rules to map fwmark → routing table
#   - Routes in each table pointing to tenant gateway
#
# Tenant Configuration:
#   Tenant A: fwmark 0x10 → Gateway A (10.10.10.107)
#   Tenant B: fwmark 0x20 → Gateway B (10.10.10.154)
#
# Author: Mikhail [azalio] Petrov
# Date: 2025

set -euo pipefail

# Configuration (can be overridden via environment variables)
# Two separate router VMs for clear traffic separation demo:
# - router1 (Gateway A): Receives Tenant A traffic
# - router2 (Gateway B): Receives Tenant B traffic
GATEWAY_A="${GATEWAY_A:-10.10.10.131}"  # router1 IP - Gateway for Tenant A
GATEWAY_B="${GATEWAY_B:-10.10.10.184}"  # router2 IP - Gateway for Tenant B
# Use fwmarks 0x10/0x20 to avoid conflict with Cilium's 0x200/0xf00 mask
FWMARK_TENANT_A="${FWMARK_TENANT_A:-0x10}"
FWMARK_TENANT_B="${FWMARK_TENANT_B:-0x20}"
TABLE_TENANT_A="${TABLE_TENANT_A:-100}"
TABLE_TENANT_B="${TABLE_TENANT_B:-200}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "Setting up tenant routing tables..."

# Add routing table names to /etc/iproute2/rt_tables
if ! grep -q "tenant-a" /etc/iproute2/rt_tables 2>/dev/null; then
    echo "${TABLE_TENANT_A} tenant-a" >> /etc/iproute2/rt_tables
    log "Added routing table 'tenant-a' (${TABLE_TENANT_A})"
fi

if ! grep -q "tenant-b" /etc/iproute2/rt_tables 2>/dev/null; then
    echo "${TABLE_TENANT_B} tenant-b" >> /etc/iproute2/rt_tables
    log "Added routing table 'tenant-b' (${TABLE_TENANT_B})"
fi

# Create IP rules for fwmark → routing table
log "Creating IP rules for fwmark routing..."

# Rule for Tenant A: fwmark 0x10 → table tenant-a
# Use word boundary to avoid matching 0x100 when checking for 0x10
if ! ip rule show | grep -qE "fwmark ${FWMARK_TENANT_A}( |$)"; then
    ip rule add fwmark ${FWMARK_TENANT_A} table tenant-a priority 50
    log "Added rule: fwmark ${FWMARK_TENANT_A} → table tenant-a"
else
    log "Rule for fwmark ${FWMARK_TENANT_A} already exists"
fi

# Rule for Tenant B: fwmark 0x20 → table tenant-b
# Use word boundary to avoid false positives
if ! ip rule show | grep -qE "fwmark ${FWMARK_TENANT_B}( |$)"; then
    ip rule add fwmark ${FWMARK_TENANT_B} table tenant-b priority 50
    log "Added rule: fwmark ${FWMARK_TENANT_B} → table tenant-b"
else
    log "Rule for fwmark ${FWMARK_TENANT_B} already exists"
fi

# Add routes in tenant routing tables (using replace for idempotency)
log "Adding routes to tenant routing tables..."

# Tenant A → Gateway A
ip route replace default via ${GATEWAY_A} table tenant-a
log "Set route: default via ${GATEWAY_A} in table tenant-a"

# Tenant B → Gateway B
ip route replace default via ${GATEWAY_B} table tenant-b
log "Set route: default via ${GATEWAY_B} in table tenant-b"

# Display configuration
log ""
log "=== Tenant Routing Configuration ==="
log ""
log "IP Rules:"
ip rule show | grep -E "fwmark|tenant"
log ""
log "Tenant A Routing Table (fwmark ${FWMARK_TENANT_A} → Gateway A ${GATEWAY_A}):"
ip route show table tenant-a
log ""
log "Tenant B Routing Table (fwmark ${FWMARK_TENANT_B} → Gateway B ${GATEWAY_B}):"
ip route show table tenant-b
log ""
log "Tenant routing setup completed!"
