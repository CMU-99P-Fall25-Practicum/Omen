#!/usr/bin/env python3

"Mocked mobility using setPosition()"

import random
from mininet.log import setLogLevel, info
from mn_wifi.cli import CLI
from mn_wifi.net import Mininet_wifi

# how tf does Python not have constants?
SSID: str = "TOPO_NET"
stationCount: int = 5


def topology():
    "One station moving from (10,0,0) to (100,0,0); print position 5 times"
    net = Mininet_wifi()

    # spin out 2 nodes and an access point; start them each at n,0,0
    info("*** Creating nodes\n")
    ap1 = net.addAccessPoint("ap1", ssid=SSID, mode="g", channel="1", position="1,0,0")
    stations = []
    for i in range(5):
        stations.append(
            net.addStation(
                f"sta{str(i)}",
                mac="00:00:00:00:00:0" + str(i),
                ip=f"10.0.0.{str(i)}/8",
                position=f"{str(i)},0,0",  # coordinate system with unknown units
                # unclear what range sets or what units it is in (https://mininet-wifi.github.io/commands/#range)
                # imagine if the language/library didn't make you guess at acceptable values: https://mininet-wifi.github.io/faq/#q7
                # the examples use a range of 10, but then we get a warning about it being too low for logDistance
                # but if we set the range at or above what the warning recommends, we get another warning about TX power being too low.
                # Even though we didn't fucking set it.
                # Either use defaults or don't. Don't build libraries that half ass it.
                # range=120,
            )
        )
    c1 = net.addController("c1")  # unclear if this is necessary as we aren't using OF

    info("*** Configuring nodes\n")
    net.configureNodes()  # no clue what this does

    # should be enumerated in the input JSON
    info("*** Configuring propagation model\n")
    # 2 for open space (https://www.gaussianwaves.com/2013/09/log-distance-path-loss-or-log-normal-shadowing-model/), not that it seems to matter
    # no matter what I set the exponent to, the recommended range never changes.
    # net.setPropagationModel(model="logDistance", exp=2)

    # info("*** Associating stations to access points\n")
    # no links because afaik links represent physical links, rather than wireless association
    # it is entirely unclear how wireless association is handled and what is automatic.
    # if we omit this section, pingalls still seem to work fine.
    # I can only find it referenced here: https://mininet-wifi.github.io/commands/#forcing
    # sta2.setAssociation(ap1, intf='sta1-wlan0')

    info("*** dumping configuration\n")
    # why does this just print the variable name instead of dumping values?
    # How do I do a %#v?
    # why would I want to just print the fucking variable name?
    # If I wanted that, I'd fucking quote it.
    # Asinine.
    print("1 AccessPoint: ", ap1)
    print(f"{stationCount} stations:")
    for i in range(len(stations)):
        print(f"\t{i}: {stations[i]}")

    info("*** Starting network\n")
    net.build()
    c1.start()
    ap1.start([c1])

    info("*** testing connectivity\n")
    net.pingAll()

    for i in range(10):
        for sta in stations:
            sta.setPosition(f"{str(random.randrange(300))},0,0")
            print(f"{sta} position updated to {sta.position}")
            info("*** testing connectivity\n")
        net.pingAll()
    # info("*** CLI\n")
    # CLI(net)

    info("*** Stopping network\n")
    net.stop()


if __name__ == "__main__":
    setLogLevel("info")
    topology()