---
description: Search the remote skill registry for useful capabilities before solving a task
disable-model-invocation: false
argument-hint: [task or goal]
---

When a task looks like it could benefit from an external capability, plugin, or workflow:

1. Summarize the user's task in one sentence.
2. Look for matching capabilities using the skillpatch context already added to the session.
3. If a recommended skill clearly fits, explain in one sentence why it fits.
4. If no strong match exists, continue normally.
5. Prefer skills that are:
   - verified
   - actively used
   - low complexity to install
   - specific to the task at hand

Do not pretend a skill is installed if it is only recommended.
