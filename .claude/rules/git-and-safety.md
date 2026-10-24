# Git and safety

- Work on feature branches; small commits; imperative messages.
- Never push to main, force push, or rewrite shared history.
- Run tests before every commit. No `--no-verify`.
- Never commit `.env*`, keys, or customer data. If found in a diff, stop and tell me.
- Database changes: write a migration, show me the SQL, wait for approval.
