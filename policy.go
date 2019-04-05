package main

import (
    "net"
    "log"
    "errors"
    "strconv"
)

var (
    ErrPolicyEmptyMatch   = errors.New("Match criteria is not defined")
    ErrPolicyEmptyAction  = errors.New("Action is node defined")
    ErrPolicyIndexInvalid = errors.New("Index out of bound")
)

type PolicyMatch struct {
    dstSubnet	*net.IPNet
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

type PolicyTable []PolicyEntry

type Policy struct {
    table PolicyTable
}

func (p *Policy) Init() {
    p.table = make(PolicyTable,0)
}

func (p *Policy) Flush() {
    p.table = nil
    p.Init()
}

func (p *Policy) Append(entry PolicyEntry) (error) {
    // TODO: error handling
    p.table = append(p.table, entry)
    return nil
}

func (p *Policy) Add(entry PolicyEntry, index int) (error) {
    // TODO: error handling
    if (index < 0) || (index >= len(p.table)) {
        return ErrPolicyIndexInvalid
    }

    p.table[index] = entry
    return nil
}

func (p *Policy) Del(index int) {
    //TODO: error handling
    p.table = append(p.table[:index], p.table[:index+1]...)
}

// TODO: other field for lookup
func (p *Policy) Lookup(pkt *Packet) (PolicyAction, bool) {
    for _, entry := range p.table {
        if p.Match(entry, pkt) {
            return entry.Action, true
        }
    }

    return PolicyAction{}, false
}

func (p *Policy) Match(entry PolicyEntry, pkt *Packet) (bool) {

    if (entry.Match.dstSubnet != nil) && (!entry.Match.dstSubnet.Contains(pkt.GetDestinationIPv4())) {
        return false
    }

    if (entry.Match.srcSubnet != nil) && (!entry.Match.srcSubnet.Contains(pkt.GetSourceIPv4())) {
        return false
    }

    return true
}

func (p *Policy) Dump() {
    var output string

    for _, item := range p.table {

        output = " "

        if item.Match.srcSubnet != nil {
            output = output + item.Match.srcSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " "

        if item.Match.dstSubnet != nil {
            output = output + item.Match.dstSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " " + strconv.Itoa(int(item.Action.egress)) + " "

        if item.Action.endpoint != nil {
            output = output + item.Action.endpoint.String()
        } else {
            output = output + "*"
        }

        output = output + " "
        output = output + strconv.Itoa(item.TimeToLive)

        log.Println(output)
    }
}

func (p *Policy) CompilePolicies(policies []PolicyEntryFile) error {

    index := 0

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
				log.Println (err)
				endpoint = nil
			}
		}

        entry.Action.egress = pol.Egress
        entry.Action.endpoint = endpoint

		p.Append(entry)
		index++
	}

    return nil
}
