/*
@Author : lsne
@Date : 2021-08-04 16:10
*/

package redisdao

type ClusterNode struct {
	ClusterID string
	Host      string
	Port      uint16
	Role      string
	Fail      bool
	MasterID  string
	Connected string
}

func GetConnected(nodes []ClusterNode) []ClusterNode {
	var n []ClusterNode
	for _, node := range nodes {
		if node.Connected == "connected" && !node.Fail {
			n = append(n, node)
		}
	}
	return n
}

func GetDisConnected(nodes []ClusterNode) []ClusterNode {
	var n []ClusterNode
	for _, node := range nodes {
		if node.Connected == "disconnected" || node.Fail {
			n = append(n, node)
		}
	}
	return n
}
