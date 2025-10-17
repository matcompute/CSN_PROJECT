#!/usr/bin/env python3
"""
CSN Mininet topology (device ↔ edge1/edge2 ↔ cloud)
Software-only emulation; we’ll control link params via tc/netem later.

h0: device
h1: edge1
h2: edge2
h3: cloud
s1: core switch
"""
from mininet.topo import Topo
from mininet.net import Mininet
from mininet.node import Controller, OVSController
from mininet.link import TCLink
from mininet.cli import CLI
from mininet.log import setLogLevel

class CSNTopo(Topo):
    def build(self):
        # hosts
        h0 = self.addHost('h0')  # device
        h1 = self.addHost('h1')  # edge1
        h2 = self.addHost('h2')  # edge2
        h3 = self.addHost('h3')  # cloud

        # single core switch
        s1 = self.addSwitch('s1')

        # links (we'll tune with tc/netem dynamically)
        self.addLink(h0, s1, cls=TCLink, bw=100, delay='5ms', loss=0)
        self.addLink(h1, s1, cls=TCLink, bw=1000, delay='2ms', loss=0)
        self.addLink(h2, s1, cls=TCLink, bw=1000, delay='2ms', loss=0)
        self.addLink(h3, s1, cls=TCLink, bw=1000, delay='20ms', loss=0)

def run():
    topo = CSNTopo()
    net = Mininet(topo=topo, link=TCLink, controller=OVSController, autoSetMacs=True, autoStaticArp=True)
    net.start()

    print("*** IPs:")
    for h in ['h0','h1','h2','h3']:
        print(h, net.get(h).IP())

    print("*** Example tc changes you can apply later:")
    print("  tc qdisc replace dev h0-eth0 root netem rate 20mbit delay 40ms loss 0.5%")

    CLI(net)
    net.stop()

if __name__ == "__main__":
    setLogLevel('info')
    run()
