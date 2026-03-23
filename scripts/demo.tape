Output demo.mp4
Set Shell bash
Sleep 500ms

Type 'python -m http.server 8000 &'
Enter
Sleep 1s
Enter

Type 'rekt 8000'
Enter
Sleep 2s

Type 'hyperfine "fuser 8000/tcp" "lsof -i :8000" "rekt 8000" -w 10'
Enter
Sleep 15000ms

Type 'rekt 8000 -k'  
Enter
Sleep 2s
