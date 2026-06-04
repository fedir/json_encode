# VHS tape for json_encode — https://github.com/charmbracelet/vhs
# Regenerate with:  vhs demo/demo.tape   (run from the repo root; needs ./json_encode built)

Output demo/json_encode.gif

Set Shell "bash"
Set FontSize 18
Set Width 1180
Set Height 760
Set Theme "Dracula"
Set TypingSpeed 13ms
Set Padding 22
Set Margin 16
Set BorderRadius 10

# Off-camera setup: binary on PATH, C locale so numbers use dots.
Hide
Type `export PATH="$PWD:$PATH"; export LC_ALL=C; clear`
Enter
Show

Sleep 400ms
Type `df -h | grep '^/dev/' | head -3 | json_encode -c -f 1,2,5 -n disk,size,used`
Enter
Sleep 1.1s

Type `ps -A -o pid,ucomm,rss | sort -k3 -rn | head -3 | json_encode -n pid,cmd,rss_kb`
Enter
Sleep 1.2s

Type `kubectl get pods -n default -o custom-columns=POD:.metadata.name,STATUS:.status.phase --no-headers | head -3 | json_encode -n pod,status`
Enter
Sleep 1.3s

Type `podman images --format '{{.Repository}}|{{.Tag}}|{{.Size}}' | grep -E 'prom/|registry' | head -3 | json_encode -c -d '|' -n image,tag,size`
Enter
Sleep 1.3s

Type `kubectl get pods -A -o custom-columns=STATUS:.status.phase --no-headers | json_encode | jq -c 'group_by(.)|map({status:.[0],count:length})'`
Enter
Sleep 1.8s
