import socket
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setblocking(0)
sock.sendto('{"format":"json","version":1}\nx'.encode("utf8"), ("127.0.0.1", 2000))