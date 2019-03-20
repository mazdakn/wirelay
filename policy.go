package main

import (
    "net"
    "log"
    "errors"
    "io/ioutil"
    "encoding/json"
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
    ingress		uint8
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
    configFile string
    Interfaces []NetIO
}

type PolicyEntryFile struct {
    Ingress		string `json:"ingress"`
    DstSubnet	string `json:"dst"`
    SrcSubnet   string `json:"src"`
    Egress		string `json:"egress"`
    Endpoint	string `json:"endpoint"`
    ttl         int    `json:"ttl"`
}

func (p *Policy) Init() {
    p.table = make(PolicyTable,0)
}

func (p *Policy) Flush() {
    p.table = nil
    p.Init()
}

func (p *Policy) CheckEntry(entry PolicyEntry) (error) {
    //TODO: need debugging

    if entry.Action.egress == NetIO_ANY {
        return ErrPolicyEmptyAction
    }

    return nil
}

func (p *Policy) Append(entry PolicyEntry) (error) {
    // TODO: error handling
    if err := p.CheckEntry(entry); err != nil {
        return err
    }

    p.table = append(p.table, entry)
    return nil
}

func (p *Policy) Add(entry PolicyEntry, index int) (error) {
    // TODO: error handling
    if (index < 0) || (index >= len(p.table)) {
        return ErrPolicyIndexInvalid
    }

    if err := p.CheckEntry(entry); err != nil {
        return err
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

    if (entry.Match.ingress != NetIO_ANY) && (entry.Match.ingress != pkt.Ingress) {
        return false
    }

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

        output = strconv.Itoa(int(item.Match.ingress)) + " "

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

func (p *Policy) CompilePoliciesJSON() error {
    var bytes []byte
    var err error
    var policies []PolicyEntryFile

    index := 0

    if bytes, err = ioutil.ReadFile(p.configFile); err != nil {
        return (err)
    }

    if err := json.Unmarshal(bytes, &policies); err != nil {
        return err
    }

    for _, pol := range policies {
        var subnet *net.IPNet
        var dev uint8
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

        dev = NetIO_ANY
        if pol.Ingress == "local" {
            dev = NetIO_LOCAL
        }

        if pol.Ingress == "tunnel" {
            dev = NetIO_TUNNEL
        }

        //TODO: error handling
        entry.Match.ingress = dev

        if pol.ttl > 0 {
            entry.TimeToLive = pol.ttl
        }

        dev = NetIO_ANY


        //TODO: change to switch
        if pol.Egress == "local" {
            dev = NetIO_LOCAL
        }

        if pol.Egress == "tunnel" {
            dev = NetIO_TUNNEL
        }

        if pol.Egress == "null" {
            dev = NetIO_NULL
        }

		if pol.Endpoint != "" {
			if endpoint, err = net.ResolveUDPAddr("udp4", pol.Endpoint); err != nil {
				log.Println (err)
				endpoint = nil
			}
		}

		switch pol.Egress {
		case "local":
			entry.Action.egress = NetIO_LOCAL
        case "drop":
            entry.Action.egress = NetIO_NULL
        case "tunnel":
            entry.Action.egress = NetIO_TUNNEL
            entry.Action.endpoint = endpoint
        default:
			log.Println ("Invalid Action - igonoring the policy")
			continue
		}

		p.Append(entry)
		index++
	}

    return nil
}
