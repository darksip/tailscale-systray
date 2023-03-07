package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/netip"
	"sort"
	"strings"
	"time"

	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
)

type ExitNode struct {
	Ip       string
	IsActive bool
	Latency  float64
}

type IpLat struct {
	Ip      string
	Latency float64
}

var (
	activeExitNode          = ""
	exitNodes               []ExitNode
	latencies               map[string][]float64
	movLatencies            map[string]float64
	nping                   int
	wantsToDisableExitNodes = false
)

func refreshExitNodes() {

	getStatus := localClient.Status
	status, err := getStatus(context.TODO())
	if err == nil {
		exitNodes = []ExitNode{}
		log.Printf("---------------------------------- nping: %d", nping)
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

func checkActiveNodeAndSetExitNode() {
	bestExitNode := getBestExitNodeIp()
	log.Printf("best exit node : %s", bestExitNode)
	log.Printf("active exit node : %s", activeExitNode)
	if !isStillActive(activeExitNode) {
		log.Printf("ouch! activeExitNode is unreachable ! let' choose another one")
		setExitNode()
		showOrderedExitNode(bestExitNode)
	} else {
		if nping > npingsCheck {
			// TODO : demand at least 30% best in latency to change
			if bestExitNode != activeExitNode {
				setExitNode()
				showOrderedExitNode(bestExitNode)
			}
		}
	}
}

func removeExitNode() {
	p, _ := localClient.GetPrefs(context.TODO())
	if p.ExitNodeID.IsZero() {
		log.Printf("No exit Node to remove")
		return
	}
	p.ClearExitNode()
	pmp := new(ipn.MaskedPrefs)
	pmp.Prefs = *p
	pmp.ExitNodeIDSet = true

	np, err := localClient.EditPrefs(context.TODO(), pmp)
	if err != nil {
		log.Printf("%s", err.Error())
	}
	if np.ExitNodeID.IsZero() {
		activeExitNode = ""
		log.Println("exit node disabled")
	}
}

func checkLatency() string {
	var bestLatency float64 = math.MaxFloat64
	var bestExitNodeIp string = ""
	nping++ // nb of ping since laste exitNode change
	for i := range exitNodes {
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
			log.Printf("%s : %f   [%f] ", exitNodes[i].Ip, exitNodes[i].Latency, movLatencies[ip])
			if movLatencies[exitNodes[i].Ip] < bestLatency {
				bestExitNodeIp = exitNodes[i].Ip
				bestLatency = movLatencies[exitNodes[i].Ip]
			}
		}
	}
	return bestExitNodeIp
}

func getBestExitNodeFromLatency() *ExitNode {
	var bestNode *ExitNode = nil
	var bestLatency float64 = math.MaxFloat64

	for i := range exitNodes {
		if latency, ok := movLatencies[exitNodes[i].Ip]; ok && latency < bestLatency {
			bestNode = &exitNodes[i]
			bestLatency = latency
		}
	}

	return bestNode
}

func getBestExitNodeIp() string {
	minLatency := 0.5
	bestIp := ""
	if nping > npingsCheck {
		bestExitNodePtr := getBestExitNodeFromLatency()
		if bestExitNodePtr != nil {
			bestExitNode := *bestExitNodePtr
			if bestExitNode.Latency > 0 && bestExitNode.Latency < minLatency {
				return bestExitNode.Ip
				//return "100.64.0.11"
			}
		}
	}
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

func forceExitNode(exitNode string) {
	if len(exitNode) > 0 {
		st, err := localClient.Status(context.TODO())
		if err != nil {
			log.Printf("%s", err.Error())
		}

		p, _ := localClient.GetPrefs(context.TODO())
		log.Printf("best exit node : %s", exitNode)
		// set exit and allow lan access local
		checkExitNodeConnection(exitNode)

		pmp := new(ipn.MaskedPrefs)
		pmp.Prefs = *p
		pmp.ExitNodeIDSet = true
		pmp.SetExitNodeIP(exitNode, st)
		pmp.ExitNodeIPSet = true
		pmp.ExitNodeAllowLANAccess = true
		pmp.ExitNodeAllowLANAccessSet = true

		np, err := localClient.EditPrefs(context.TODO(), pmp)
		if err != nil {
			log.Printf("%s", err.Error())
		}
		if np.ExitNodeIP.IsValid() {
			activeExitNode = exitNode
			nping = 0
		}
	}
}

func setExitNode() {
	if noExitNode > 0 {
		return
	}
	refreshExitNodes()
	exitNode := checkLatency()
	//exitNode := getBestExitNodeIp()
	forceExitNode(exitNode)
}

func setExitNodeOff() {
	if len(activeExitNode) > 0 {
		removeExitNode()
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

func showOrderedExitNode(ben string) {
	// get five first
	var ipsl []IpLat
	for ip, lat := range movLatencies {
		ipsl = append(ipsl, IpLat{ip, lat})
	}
	sort.Slice(ipsl, func(i, j int) bool {
		return ipsl[i].Latency < ipsl[j].Latency
	})

	for i, ipl := range ipsl {
		if i > 5 {
			break
		}
		id := fmt.Sprintf("EN%d", i+1)
		label := fmt.Sprintf("%-16s[%8.2f ms]", ipl.Ip, ipl.Latency*1000)
		sm.SetLabel(id, label)
		sm.SetHidden(id, false)
		if ipl.Ip == activeExitNode {
			sm.SetIcon(id, "bluearrow")
		} else {
			if ipl.Ip == ben {
				sm.SetIcon(id, "greyarrow")
			} else {
				sm.SetIcon(id, "empty")
			}
		}

	}
}

func getExitNodeIpForId(id string) string {
	elt, _ := sm.GetById(id)
	t := strings.Split(elt.Label, " ")
	return t[0]
}

func AddExitNodeHandlersToMenu() {
	sm.SetHandler("EXITNODE_ON", func() {
		wantsToDisableExitNodes = false
		setExitNode()
		Notify("Exit node is active, your traffic is protected", "exitnode")
	})
	sm.SetHandler("EXITNODE_OFF", func() {
		wantsToDisableExitNodes = true
		Notify("Exit node deactivated, your traffic is unprotected", "exitnode")
		//sm.SetDisabled("EXITNODE_OFF", true)
	})
	sm.SetHandler("EN1", func() { forceExitNode(getExitNodeIpForId("EN1")) })
	sm.SetHandler("EN2", func() { forceExitNode(getExitNodeIpForId("EN2")) })
	sm.SetHandler("EN3", func() { forceExitNode(getExitNodeIpForId("EN3")) })
	sm.SetHandler("EN4", func() { forceExitNode(getExitNodeIpForId("EN4")) })
	sm.SetHandler("EN5", func() { forceExitNode(getExitNodeIpForId("EN5")) })
}
