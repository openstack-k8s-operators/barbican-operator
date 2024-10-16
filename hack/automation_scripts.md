# Automation Scripts

## Build Custom Container Images

This script is meant to be used when the environment where Barbican will work has an HSM (Hardware Security Module) in place.
Since Barbican is able to store secrets when an HSM is present, it needs to be made aware of such presence. Two components
are directly affected by the presence of an HSM:  Barbican API and Barbican Worker. The default container images do not have
any HSM client software embedded in them since each manufacturer may have specifics on how their software can be distributed.

Therefore, the container images need to be customized to include such client software. Depending on the vendor, the installation
procedures might differ. You need to run the script corresponding to the vendor you use.

### Eviden Trustway (previously, ATOS)

For the Eviden (previously, ATOS) Trustway HSMs, you will use the `build_custom_image-eviden.sh` script. The usage is as follows:

```bash
$ bash build_custom_image-eviden.sh <source_registry_host> <namespace> <barbican-api_image_tag> <barbican-worker_image_tag> <eviden_iso_file> <destination_registry_host>
```

where:
* `source_registry_host`:  corresponds to the FQDN (Fully Qualified Domain Name) of the registry that holds the default container images.
Example:  quay.io.
* `namespace`:  it's an internal repository organization that matches the OpenStack distribution with an operating system. Example: `podified-antelope-centos9`.
* `barbican-api_image_tag`:  because OpenStack container images may not have the usual `latest` tag, you may need to manually obtain and provide the newest tag.  Example:  `75c508097e39a3423d9f2eef86648c4e`.
* `barbican-worker_image_tag`:  something similar happens for the Barbican Worker image.  Example:  `71849c7583fa95ee18dcc0c73c93569d`.
* `eviden_iso_file`:  this is the filename of the ISO file holding the Eviden HSM client software.  Example:  `Proteccio3.00.03.iso`.  **Please put it in the same directory as this script.**
* `destination_registry_host`:  corresponds to the FQDN (Fully Qualified Domain Name) of the registry that will store the customized container images.
Example:  hub.docker.com.

>**Note 1**<br>
**You need to edit the `build_custom_image-eviden.sh` script to include your username on the container registry. <br> This is necessary since one of the final steps the script takes is to push the new customized image to the registry.**

>**Note 2**<br>
**You must install both, `podman` and `buildah` for this script to successfully execute.**

The script proceeds as follows:
1. Pulls with `podman` the container images of Barbican API and Barbican Worker from the provided registry host.
2. Creates a temporary directory and mounts the ISO client file on it.
3. Creates two `Dockerfile`s (`Dockerfile.barbican-api` and `Dockerfile.barbican-worker`), for the two new container images. These files have instructions to copy and invoke the Eviden's installation script.
4. Builds the new images with `buildah`.  At this stage, temporary containers will be created to execute the client software installation.
5. Pushes the new image to the registry host you provided as parameter.  **For this sake, the script needs to be edited to have your username on this registry host.**
6. Unmounts the ISO client software and deletes the temporary directory created on step 2.

After this script successfully runs, you can start your Barbican operator deployment.
