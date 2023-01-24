
# BookInfo application Test
This test set [Istio BookInfo application](https://istio.io/latest/docs/examples/bookinfo/) on the host and destination clusters. The Product and details microservices run on the host cluster, and the Reviews and Rating microservices run on the destination clusters. 
   * To Build the kind clusters and run the bookInfo application:    
```make run-kind-bookinfo```  
   * The BookInfo application can be viewed by connecting to the Product microservice:  
```firefox http://<host-kind-cluster-ip>:30001/productpage```
