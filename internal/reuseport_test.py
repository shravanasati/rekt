import socket, os, sys, time

PORT = 8000
def worker(n):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEPORT, 1)
    s.bind(('', PORT))
    s.listen(1)
    print(f"worker {n} PID={os.getpid()} listening")
    time.sleep(999)

count = int(sys.argv[1]) if len(sys.argv) > 1 else 3
pids = []
for i in range(count):
    pid = os.fork()
    if pid == 0:
        worker(i)
        sys.exit(0)
    pids.append(pid)

print(f"spawned {count} workers: {pids} on port {PORT}")
print("press enter to kill all")
input()
for pid in pids:
    os.kill(pid, 9)