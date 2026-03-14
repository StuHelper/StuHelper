<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

Use the `/trellis:start` command when starting a new session to:
- Initialize your developer identity
- Understand current project context
- Read relevant guidelines

Use `@/.trellis/` to learn:
- Development workflow (`workflow.md`)
- Project structure guidelines (`spec/`)
- Developer workspace (`workspace/`)

Keep this managed block so 'trellis update' can refresh the instructions.

<!-- TRELLIS:END -->

## Required Session Start

Before replying to the first user message in a new session, read these files in order:

1. `/Users/zxy/Code/StuHelper/.trellis/workflow.md`
2. `/Users/zxy/Code/StuHelper/.trellis/spec/guides/index.md`
3. `/Users/zxy/Code/StuHelper/.trellis/spec/backend/index.md`
4. `/Users/zxy/Code/StuHelper/.trellis/spec/frontend/index.md`
5. `/Users/zxy/Code/StuHelper/.trellis/workspace/index.md`
6. Run `python3 ./.trellis/scripts/get_developer.py` from `/Users/zxy/Code/StuHelper` to resolve the active developer ID
7. `/Users/zxy/Code/StuHelper/.trellis/workspace/<developer>/index.md` using the resolved developer ID
8. `/Users/zxy/Code/StuHelper/.trellis/workspace/<developer>/journal-1.md` for the `Project Archive` section, plus the active journal file if it is different

If the task is clearly backend-only or frontend-only, you should still skim both backend and frontend indexes first, then go deeper into the relevant side.

## Required Working Agreements

- Use Chinese for user-facing communication and code comments when comments are needed.
- Keep identifiers, filenames, and commit messages in English.
- Commit messages must follow Conventional Commits and must not include AI vendor branding.
- Treat the project as pre-launch by default: prefer the most thorough, enterprise-grade design instead of migration-minimizing patches.
- Do not commit secrets, hardcode config values, ignore errors silently, use `any` without strong justification, or keep oversized functions when they should be split.

## Required Recording Rules

- After meaningful code or documentation changes, update the relevant Trellis record instead of any legacy rule file.
- Project-wide architecture, workflow, or convention changes must be reflected in `journal-1.md` under the `Project Archive` section.
- After implementation is committed, record the session with `python3 ./.trellis/scripts/add_session.py`.
