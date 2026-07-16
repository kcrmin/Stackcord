# Lifecycle reference

Use dependency gates, not waterfall scheduling:

entry diagnosis → service discovery → project initialization → complete product definition → architecture and technology selection → complete executable UI baseline → contracts and DBML → stable implementation boundaries → vertical full-stack implementation → integration → production hardening → release candidate → user validation → release → operations.

Reopen an earlier gate when later evidence invalidates it. Split large UI baselines by role, domain, and journey while keeping one approved product-wide coverage model. Define shared interfaces and failure behavior before parallel implementations, then deliver vertical slices with TDD.
