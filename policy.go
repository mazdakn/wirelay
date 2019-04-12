package main

import (
    "log"
    "net"
    "errors"
    "strconv"
)

var (
    ErrPolicyEmptyMatch   = errors.New("Match criteria is not defined")
    ErrPolicyEmptyAction  = errors.New("Action is node defined")
    ErrPolicyIndexInvalid = errors.New("Index out of bound")
)

type PolicyMatch struct {
    dstSubnet   *net.IPNet
    srcSubnet   *net.IPNet
}

type PolicyAction struct {
    egress      uint8
    endpoint    *net.UDPAddr
}

type PolicyEntry struct {
    Match  PolicyMatch
    Action PolicyAction
    TimeToLive  int
}

type Policy struct {
	rules []PolicyEntry
}

func (p *Policy) CompilePolicies(policies []PolicyEntryFile) (error) {

    tmpPolicy := make([]PolicyEntry, 0)

    for _, pol := range policies {
        var subnet *net.IPNet
        var endpoint *net.UDPAddr
        var err error

        entry := PolicyEntry{}

        if pol.DstSubnet != "" {
            if _, subnet, err = net.ParseCIDR(pol.DstSubnet); err != nil {
                // TODO: log and continue
                return err
            }

            entry.Match.dstSubnet = subnet
        }

        if pol.SrcSubnet != "" {
            if _, subnet, err = net.ParseCIDR(pol.SrcSubnet); err != nil {
                //TODO: log and continue
                return err
            }

            entry.Match.srcSubnet = subnet
        }

        if pol.ttl > 0 {
            entry.TimeToLive = pol.ttl
        }

        if pol.Endpoint != "" {
            if endpoint, err = net.ResolveUDPAddr("udp4", pol.Endpoint); err != nil {
                return err
            }
        }

        switch pol.Action {
        case "LOCAL"    : entry.Action.egress = NETIO_LOCAL  //e.local.netio
        case "FORWARD"  : entry.Action.egress = NETIO_FORWARD //e.tunnel.netio
        default         : entry.Action.egress = NETIO_DROP //nil
        }

        entry.Action.endpoint = endpoint

        tmpPolicy = append(tmpPolicy, entry)
    }

    p.rules = tmpPolicy
    return nil
}


func (p *Policy) Lookup(pkt *Packet) (PolicyAction, bool) {

    for _, entry := range p.rules {
        if (entry.Match.dstSubnet != nil) && (!entry.Match.dstSubnet.Contains(pkt.GetDestinationIPv4())) {
            continue
        }

        if (entry.Match.srcSubnet != nil) && (!entry.Match.srcSubnet.Contains(pkt.GetSourceIPv4())) {
            continue
        }

        return entry.Action, true
    }

    return PolicyAction{}, false
}


func (p *Policy) DumpPolicies() {
    var output string

    Print("Engine policies:")

    for index, pol := range p.rules {

        output = "[" + strconv.Itoa(index) + "] "

        if pol.Match.srcSubnet != nil {
            output = output + pol.Match.srcSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " "

        if pol.Match.dstSubnet != nil {
            output = output + pol.Match.dstSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " ==> "

        switch pol.Action.egress {
        case NETIO_LOCAL   : output = output + "local "
        case NETIO_FORWARD : output = output + "forward "
        case NETIO_DROP    : output = output + "drop "
        default            : output = output + "unknown "
        }

        if pol.Action.endpoint != nil {
            output = output + pol.Action.endpoint.String()
        }

        if pol.TimeToLive != 0 {
            output = " " + output + strconv.Itoa(pol.TimeToLive)
        }

        log.Println(output)
    }
}
