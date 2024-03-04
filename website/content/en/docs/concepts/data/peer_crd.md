type Peer struct {
  metav1.TypeMeta  
  metav1.ObjectMeta // Peer name must match the Subject name presented in its certificate

  Spec   PeerSpec
  Status PeerStatus
}

type PeerSpec struct {
  Gateways []string // one or more gateway addresses, each in the form "host:port"
  Attributes map[string]string // Peer's attribute set
  // TODO: should we have fixed/required set of attributes explicitly called out and
  // an optional attribute set encoded as a map?)
}
 
type PeerStatus struct {
  ObservedGeneration int64
  Conditions[] metav1.Condition
}
