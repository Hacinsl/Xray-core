package net

func (n Network) SystemString() string {
	switch n {
	case Network_TCP:
		return "tcp"
	case Network_UDP:
		return "udp"
	case Network_UNIX:
		return "unix"
	default:
		return "unknown"
	}
}

// HasNetwork returns true if the network list has a certain network.
func HasNetwork(list []Network, network Network) bool {
	for _, value := range list {
		if value == network {
			return true
		}
	}
	return false
}

// HasDelivery returns true if the delivery list has a certain delivery.
func HasDelivery(list []Delivery, delivery Delivery) bool {
	for _, value := range list {
		if value == delivery {
			return true
		}
	}
	return false
}

// ToDelivery converts a Network, which describes an address or socket type,
// to the Delivery kind describing how a connection is handed to Process().
func (n Network) ToDelivery() Delivery {
	switch n {
	case Network_TCP:
		return Delivery_Stream
	case Network_UDP:
		return Delivery_Packet
	case Network_UNIX:
		return Delivery_Unix
	default:
		return Delivery_Unspecified
	}
}

// ToNetwork converts a Delivery kind to the Network it corresponds to.
func (d Delivery) ToNetwork() Network {
	switch d {
	case Delivery_Stream:
		return Network_TCP
	case Delivery_Packet:
		return Network_UDP
	case Delivery_Unix:
		return Network_UNIX
	default:
		return Network_Unknown
	}
}

// ToDeliveries converts a list of Networks to Delivery kinds.
func ToDeliveries(list []Network) []Delivery {
	deliveries := make([]Delivery, 0, len(list))
	for _, network := range list {
		deliveries = append(deliveries, network.ToDelivery())
	}
	return deliveries
}
