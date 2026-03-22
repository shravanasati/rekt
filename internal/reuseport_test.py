import socket, os, sys, time
import multiprocessing

PORT = 8000
def worker(n):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEPORT, 1)
    except AttributeError:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(('', PORT))
    s.listen(1)
    print(f"worker {n} PID={os.getpid()} listening")
    try:
        time.sleep(999)
    except KeyboardInterrupt:
        pass

if __name__ == '__main__':
    count = int(sys.argv[1]) if len(sys.argv) > 1 else 3
    processes = []
    for i in range(count):
        p = multiprocessing.Process(target=worker, args=(i,))
        p.start()
        processes.append(p)

    print(f"spawned {count} workers: {[p.pid for p in processes]} on port {PORT}")
    print("press enter to kill all")
    try:
        input()
    except (KeyboardInterrupt, EOFError):
        pass
    for p in processes:
        p.terminate()
        p.join()