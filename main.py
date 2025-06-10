from scapy.all import *
import argparse
import random

parser = argparse.ArgumentParser(description="Send TCP SYN packets one by one and check response.")
parser.add_argument("--sport", type=int, help="Source port number (0-65535). Optional. Random if omitted.")
parser.add_argument("--count", type=int, default=1, help="Number of packets to send")
parser.add_argument("--dst", type=str, required=True, help="Destination IP address or hostname")
parser.add_argument("--dport", type=int, required=True, help="Destination port")

args = parser.parse_args()

for i in range(args.count):
    sport = args.sport if args.sport is not None else random.randint(1024, 65535)
    pkt = IP(dst=args.dst)/TCP(sport=sport, dport=args.dport, flags="S", seq=100)

    print(f"Sending to: {args.dst}:{args.dport} ... ", end="", flush=True)

    resp = sr1(pkt, timeout=1, verbose=False)

    if resp and TCP in resp and resp[TCP].flags == "SA":
        print("Done")
    else:
        print("Failed")