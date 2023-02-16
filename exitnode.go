package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"time"

	"tailscale.com/tailcfg"
)

type ExitNode struct {
	Ip       string
	IsActive bool
	Latency  float64
}

var (
	activeExitNode = ""
	exitNodes      []ExitNode
	latencies      map[string][]float64
	movLatencies   map[string]float64
	nping          int
)

func refreshExitNodes() {

	getStatus := localClient.Status
	status, err := getStatus(context.TODO())
	if err == nil {
		exitNodes = []ExitNode{}
		log.Print("----------------------------------")
		for _, ps := range status.Peer {
			if len(ps.TailscaleIPs) != 0 {
				peerIP := ps.TailscaleIPs[1].String()
				log.Printf("peer %s (%s): EN: %t ENOption: %t", ps.HostName, peerIP, ps.ExitNode, ps.ExitNodeOption)
				if ps.ExitNodeOption {
					isa := activeExitNode == peerIP
					en := ExitNode{
						Ip:       peerIP,
						IsActive: isa,
						Latency:  0.0,
					}
					exitNodes = append(exitNodes, en)
					//exitNode = peerIP
				}
			}
		}
	}
}

func checkLatency() {
	for i, _ := range exitNodes {
		ip, lat := pingExitNode(&exitNodes[i])
		if lat == 0.0 {
			log.Printf("%s : %f   [%f]", exitNodes[i].Ip, 0.0, movLatencies[ip])
			// don't add a 0.0 to the avg , we would give a bonus to a disfunctional exitNode
			continue
		}
		if len(latencies[ip]) >= 20 {
			latencies[ip] = append(latencies[ip][1:], lat)
		} else {
			latencies[ip] = append(latencies[ip], lat)
		}
		if len(latencies[ip]) > 0 {
			movLatencies[ip] = 0
			for _, l := range latencies[ip] {
				movLatencies[ip] += (l / float64(len(latencies[ip])))
			}
			log.Printf("%s : %f   [%f]", exitNodes[i].Ip, exitNodes[i].Latency, movLatencies[ip])
		}
	}
}

func getBestExitNodeIp() string {
	minLatency := 0.5
	bestIp := ""
	for _, en := range exitNodes {
		if en.Latency > 0.0 && en.Latency < minLatency {
			bestIp = en.Ip
			minLatency = en.Latency
		}
	}
	return bestIp
}

func isStillActive(active string) bool {
	for _, en := range exitNodes {
		if en.Ip == active {
			if en.Latency > 0.0 && en.Latency < 0.5 {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func setExitNode() {
	refreshExitNodes()
	checkLatency()
	exitNode := getBestExitNodeIp()
	if len(exitNode) > 0 {
		log.Printf("best exit node : %s", exitNode)
		// set exit and allow lan access local
		checkExitNodeConnection(exitNode)
		exitNodeParam := fmt.Sprintf(`--exit-node=%s`, exitNode)
		o, errset := execCommand(cliExecutable, "set", exitNodeParam)
		if errset != nil {
			log.Printf("%s", o)
			log.Printf(errset.Error())
		} else {
			activeExitNode = exitNode
			menuExitNode.SetTitle("Set Exit Node Off")
			o, errset = execCommand(cliExecutable, "set", "--exit-node-allow-lan-access")
			// reset ping count
			nping = 0
		}
	}
}

func pingExitNode(exitNode *ExitNode) (string, float64) {
	ip, err := netip.ParseAddr(exitNode.Ip)
	if err == nil {
		ctx, _ := context.WithTimeout(context.TODO(), 2*time.Second)
		res, err := localClient.Ping(ctx, ip, tailcfg.PingPeerAPI)
		if err == nil {
			if res.LatencySeconds < 3.0 {
				exitNode.Latency = res.LatencySeconds
			} else {
				res, err := localClient.Ping(context.TODO(), ip, tailcfg.PingPeerAPI)
				if err == nil {
					exitNode.Latency = res.LatencySeconds
					return exitNode.Ip, res.LatencySeconds
				}
			}
			return exitNode.Ip, res.LatencySeconds
		}
		return exitNode.Ip, 0.0
	}
	return "", 0.0
}

func checkExitNodeConnection(en string) {
	ntry := 0
	for {
		ip, err := netip.ParseAddr(en)
		if err == nil {
			log.Printf("trying to ping ExitNode %s", ip.String())
			res, err := localClient.Ping(context.TODO(), ip, tailcfg.PingPeerAPI)
			if err == nil {
				log.Printf("ping latency: %f", res.LatencySeconds)
				break
			}
			log.Println(err.Error())
			time.Sleep(3 * time.Second)
			ntry++
		} else {
			log.Println(err.Error())
			time.Sleep(3 * time.Second)
			ntry++
		}
		if ntry > 10 {
			// launch reconnect sequence and exit
			go disconnectReconnect()
			return
		}
	}

}
