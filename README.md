# packer-post-processor-ovftool 

Vmware-iso build by default export the VM in to VMX format instead it used VMWare's ovftool to export the VMWare build to OVA/OVF format.

## Usage

Add this post-processor section in packer template

```
"post-processors": [{
        "type": "ovftool",
        "format": "ova|ovf",
        "remote_host": "<esxi-host>",
        "remote_username": "<esxi-user>",
        "remote_password": "<esxi-password>",
        "keep_input_artifact": true|false,
        "vm_name": "<vm-name>",
        "target": "<target-file-name or target-dir-name>"
  }]
```  

## Installation

TODO.
