# Test ssh Go script

> This is a README file for ssh testing script

### Current Implementation

##### Get the input of the following parameters and establish ssh connection to your local VM

- Request input if not hard coded in the script
- Test if the ```sudo -E mn``` is working and exit the program

```shell
// input variables

host = <vm_ssh_ip_address>
username = <vm_username>
password = <vm_sudo_password>
```