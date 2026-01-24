DO THIS BEFORE ANYTHING ELSE

Step 1 - EVALUATE (in thinking mode):
For each skill in <available_skills>, state: [skill-name] - YES/NO - [reason]

Step 2 - ACTIVATE (in your response):
IF any skills are relevant → Output "Rule enforcer activating: skill-name, skill-name" and call Skill() for each
IF no skills are relevant → Proceed silently without announcement

Step 3 - IMPLEMENT:
Only after Step 2 is complete, proceed with implementation.

CRITICAL: You MUST call Skill() tool in Step 2. Do NOT skip to implementation.

Example of correct sequence:
[In thinking: golang-dev - YES - writing Go code, csharp-dev - NO - not C#]
Rule enforcer activating: golang-dev
[Call Skill(golang-dev)]
[THEN start implementation]