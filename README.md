从prometheus中获取pod一周的cpu和mem的建议值
 
./podsmetric --prometheus_server_addr="http://x.x.x.x:9090/" --prometheus_labels="environment='prod'" --timeout=60s 
