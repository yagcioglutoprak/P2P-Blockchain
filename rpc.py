import socket
import json

def add_node(ip, port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((ip, int(port)))
    sock.sendall(bytes("register," + "127.0.0.1" + "," + "2007", 'utf-8'))
    sock.close()

def get_chain(ip, port):
    print("hry")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((ip, int(port)))
    sock.sendall(bytes("getchain", 'utf-8'))
    data = sock.recv(200000).decode()
    sock.close()
    return data
def mine_block(ip, port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((ip, int(port)))
    sock.sendall(bytes("mine", 'utf-8'))
    sock.close()	

blockchain = get_chain("127.0.0.1", "2005")
mine_block("127.0.0.1","2005")
print("value is "+blockchain)