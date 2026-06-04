# VHS tape for json_encode — https://github.com/charmbracelet/vhs
# Regenerate with:  vhs demo/demo.tape   (run from the repo root; needs ./json_encode built)

Output demo/json_encode.gif

Set Shell "bash"
Set FontSize 19
Set Width 1100
Set Height 640
Set Theme "Dracula"
Set TypingSpeed 28ms
Set Padding 22
Set Margin 16
Set BorderRadius 10

# Put the freshly built binary on PATH without showing the setup.
Hide
Type `export PATH="$PWD:$PATH"; clear`
Enter
Show

Sleep 600ms
Type "# Any shell output -> JSON, one pipe away. Numbers are typed automatically."
Enter
Type "printf 'cpu 12\nmem 47\n' | json_encode -k"
Enter
Sleep 1.8s

Type "# A header row names the columns -> array of objects"
Enter
Type "printf 'NAME PID STATUS\nnginx 1240 up\nredis 880 up\n' | json_encode -H"
Enter
Sleep 2.3s

Type "# IDs and versions stay strings (round-trip safe)"
Enter
Type "printf 'id 007\nver 1.2.3\nport 8080\n' | json_encode -k"
Enter
Sleep 2.3s

Type "# Piped? Output goes compact and uncolored, ready for jq"
Enter
Type "printf 'a\nb\nc\n' | json_encode | cat"
Enter
Sleep 2.2s
