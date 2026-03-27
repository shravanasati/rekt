import signal, socket, os, time

signal.signal(signal.SIGTERM, signal.SIG_IGN)

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('', 8080))
s.listen(1)

print(f"PID={os.getpid()} listening on 8080, ignoring SIGTERM")
while True:
    time.sleep(1)
