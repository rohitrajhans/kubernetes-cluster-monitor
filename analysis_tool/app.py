# Retrieve logs via pull mechanism from the logger/log-api container
# pass url as command line argument
# how to get url:
# 1. kubectl config set-context --current --namespace=sidecar-injector
# 2. kubectl get svc
# 3. Check the IP and Port for logger-svc
# 4. Logs url will be: http://ip:port/get_logs

import requests
import sys
import json
import pandas as pd

n = len(sys.argv)
url = ""

for i in range(1, n):
    if sys.argv[i] == "-u":
        url = sys.argv[i+1]
        
if url == "":
    print("Pass url")
    sys.exit()

resp = requests.get(url)
json_data = json.loads(resp.text)
#print(json_data)

df = pd.DataFrame(json_data)
df = df.drop_duplicates()
print("There are total", len(df), "entries")
print(df.head())

df.to_csv("data.csv", index=False)
print("Stored data to csv")
