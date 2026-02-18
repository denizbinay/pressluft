# Technical Principles and Constraints

This document defines the non-negotiable engineering invariants of Pressluft.  
These rules exist to prevent architectural drift and protect determinism.

## System Invariants

**Determinism over cleverness**  
All operations must produce predictable outcomes. Hidden side effects, implicit mutations, and non-transactional workflows are forbidden.

**Single Source of Truth**  
The control plane database defines reality. The filesystem and remote nodes are derived state. If divergence occurs, the database wins.

**State Machine Authority**  
Every lifecycle action must pass through the explicit state machine. Direct execution paths that bypass modeled transitions are prohibited.

**One Mutation per Site**  
Concurrent destructive operations on the same site are not allowed. Concurrency control is mandatory and enforced at the persistence layer.

**Idempotent Infrastructure**  
Provisioning and configuration steps must be safely repeatable. Network interruption or partial failure must never corrupt a node.

**Isolation by Construction**  
Each site receives dedicated OS user, PHP-FPM pool, database, and web server configuration. Shared writable state across sites is forbidden.

**Immutable Releases**  
Deployments must use atomic release directories with symlink switching. In-place overwrites are disallowed.

**Agentless Nodes**  
Nodes are controlled exclusively via SSH. No resident agents, background daemons, or message brokers on managed hosts.

## Explicit Constraints

- No Kubernetes dependency  
- No Docker requirement for core operation  
- No distributed system primitives for MVP  
- No destructive promotion without verified backup  
- No feature that compromises operational simplicity  

Pressluft prioritizes boring reliability, explicit modeling, and long-term evolvability over trend-driven complexity.

