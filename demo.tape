# VHS Tape Script for Crenox Demo (v2.1.8)
# Run: vhs < demo.tape to generate demo.gif
# --------------------------------------------------

Output docs/demo.gif

# Terminal Settings
Set Shell "bash"
Set FontSize 15
Set FontFamily "JetBrains Mono"
Set Width 1200
Set Height 600
Set Padding 20
Set Theme "Dracula"

# 1. Automated One-Line Install from Documentation
Type "curl -fsSL https://crenoxhq.github.io/crenox/install.sh | bash"
Enter
Sleep 2s
Ctrl+L
Sleep 0.5s

# 2. Version Verification
Type "crenox version"
Enter
Sleep 1s

# 3. Stage safe file and commit (passes cleanly)
Type "git add app.py"
Enter
Sleep 0.5s
Type "git commit -m 'Initial commit'"
Enter
Sleep 2s
Ctrl+L
Sleep 0.5s

# 4. Stage AWS secret and commit (blocked with defensive masking)
Type "git add config.py"
Enter
Sleep 0.5s
Type "git commit -m 'Configure AWS credentials'"
Enter
Sleep 3s

# 5. Instant ad-hoc scanner
Type "crenox scan config.py"
Enter
Sleep 2s
Ctrl+L
Sleep 0.5s

# 6. Suppress with inline ignore directive & commit
Type "git add config.py"
Enter
Sleep 0.5s
Type "git commit -m 'Configure AWS credentials (suppressed)'"
Enter
Sleep 2s

# 7. Verify clean log
Type "git log --oneline"
Enter
Sleep 2s
