# importing the requests library
import requests
import logging
import argparse
import json
import http
  
# api-endpoint
api_uri = {
    'add_target' : "target/{}/{}",
    'add_labels' : 'labels/update/{}'
}

def load_sample_targets(targets_file):
    with open(targets_file) as f:
        return json.load(f)

def add_targets(args, target_list):

    for tg,tg_conf in target_list.items():
        
        for t in tg_conf['targets']:
            print(f"Adding host {t} to target group {tg}")
            url = f"{args['host']}/{api_uri['add_target'].format(tg,t)}"
            if args['debug']:
                print(f"\tRunning: POST {url}")
            r = requests.post(url = url)
        
        print(f"Adding labels to target group {tg}")
        qsargs = []
        for lbl_name,lbl_value in tg_conf['labels'].items():
            qsargs.append(f"labels={lbl_name}={lbl_value}")
        
        url = f"{args['host']}/{api_uri['add_labels'].format(tg)}?{'&'.join(qsargs)}"
        if args['debug']:
            print(f"\tRunning: POST {url}")

        r = requests.post(url = url)

        print("")


def main(args):

    if 'debug' in args and args['debug']:
        http.client.HTTPConnection.debuglevel = 1

    if args['option'] == 'add':
        target_list = load_sample_targets(args['file'])
        add_targets(args, target_list)
    else:
        
    

if __name__ == "__main__":

    parser = argparse.ArgumentParser(description='Utility to load sample targets from JSON file')
    parser.add_argument('-o','--option', dest='option', choices=['add_all', 'remove_all'], help='Add or remove all targets')
    parser.add_argument('-f','--file', dest='file', help='File containing the json targets')
    parser.add_argument('-d','--debug', dest='debug', action='store_true', help='Enable debug output')
    parser.add_argument('--host', dest='host', help='API host')
    args = vars(parser.parse_args())

    main(args)