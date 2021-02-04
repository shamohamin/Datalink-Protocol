import subprocess

def main():
    server_path = "server/binary"
    client_path = "client/binary"
    output_file = "out"
    os_list = subprocess.getoutput("go tool dist list").split('\n')
    main_path = subprocess.getoutput('pwd')
    
    for os_name in os_list:
        os, arch = os_name.split('/')
        binary_path_server = f"{main_path}/{server_path}/{os}_{arch}"
        print(binary_path_server)
        subprocess.getoutput(f"mkdir {binary_path_server}")
        subprocess.getoutput(f"cd {main_path}/server && go build -o {output_file}")
        subprocess.getoutput(f"mv {main_path}/server/out {binary_path_server}")
        
        binary_path_client = f"{main_path}/{client_path}/{os}_{arch}"
        print(binary_path_client)
        subprocess.getoutput(f"mkdir {binary_path_client}")
        subprocess.getoutput(f"cd {main_path}/client && go build -o {output_file}")
        subprocess.getoutput(f"mv {main_path}/client/out {binary_path_client}")
        
    
    


if __name__ == '__main__':
    main()
    