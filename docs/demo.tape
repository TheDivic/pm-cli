# README demo, recorded with charmbracelet/vhs (https://github.com/charmbracelet/vhs).
#
#   make demo        # from the repository root; writes docs/demo.gif
#
# The recording runs against a throwaway workspace created by
# scripts/demo-setup.sh, never against real task files. Keep the command list
# short: the point is to show what pm feels like, not to document it.

Output docs/demo.gif

Require pm

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 760
Set Padding 20
Set Margin 16
Set BorderRadius 8
Set WindowBar Colorful
Set TypingSpeed 55ms
Set Theme "Catppuccin Mocha"

# Seed a temporary workspace and quiet the prompt, off camera.
Hide
Type "export PM_ROOT=$(scripts/demo-setup.sh) && cd $PM_ROOT"
Enter
Type "export PS1='$ ' && clear"
Enter
Show

Sleep 800ms

# Projects: progress at a glance, finished ones out of the way by default.
Type "pm projects list"
Enter
Sleep 3s

# Tasks: pre-ranked, so the top row is what to do next.
Type "pm tasks list"
Enter
Sleep 3.5s

# Capture without deciding where it belongs — straight to the inbox.
Type "pm tasks add -t 'Ask legal about the cookie banner'"
Enter
Sleep 2.5s

# Move work along, several tasks at once.
Type "pm tasks status web-003 web-004 in-progress"
Enter
Sleep 2.5s

# One project in detail: fields and progress.
Type "pm projects show website"
Enter
Sleep 3s

# Its Markdown document, rendered for the terminal.
Type "pm projects doc website"
Enter
Sleep 4s

# Structured output for agents and scripts.
Type "pm --json tasks list -s todo"
Enter
Sleep 3.5s

Sleep 1s
