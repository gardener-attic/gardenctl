<%
  import os, yaml

  values={}
  if context.get("values", "") != "":
    values=yaml.load(open(context.get("values", "")))

  if context.get("cloud", "") == "":
    raise Exception("missing --var cloud={aws,azure,gcp,openstack,local} flag")

  def value(path, default):
    keys=str.split(path, ".")
    root=values
    for key in keys:
      if isinstance(root, dict):
        if key in root:
          root=root[key]
        else:
          return default
      else:
        return default
    return root

  entity=""
  if cloud == "aws":
    entity="AWS account"
  elif cloud == "azure" or cloud == "az":
    entity="Azure subscription"
  elif cloud == "gcp":
    entity="GCP project"
  elif cloud == "openstack" or cloud == "os":
    entity="OpenStack tenant"
%>---<% if entity != "": print("# Secret containing cloud provider credentials for " + entity + " into which Shoot clusters should be provisioned.") %>
apiVersion: v1
kind: Secret
metadata:
  name: ${value("metadata.name", "core-" + cloud)}
  namespace: ${value("metadata.namespace", "garden-dev")}
  labels:
    cloudprofile.garden.sapcloud.io/name: ${cloud} # label is only meaningful for Gardener dashboard
type: Opaque
data:
  % if cloud == "aws":
  accessKeyID: ${value("data.accessKeyID", "base64(access-key-id)")}
  secretAccessKey: ${value("data.secretAccessKey", "base64(secret-access-key)")}
  % endif
  % if cloud == "azure" or cloud == "az":
  tenantID: ${value("data.tenantID", "base64(tenant-id)")}
  subscriptionID: ${value("data.subscriptionID", "base64(subscription-id)")}
  clientID: ${value("data.clientID", "base64(client-id)")}
  clientSecret: ${value("data.clientSecret", "base64(client-secret)")}
  % endif
  % if cloud == "gcp":
  serviceaccount.json: ${value("data.serviceaccountjson", "base64(serviceaccount-json)")}
  % endif
  % if cloud == "openstack" or cloud == "os":
  domainName: ${value("data.domainName", "base64(domain-name)")}
  tenantName: ${value("data.tenantName", "base64(tenant-name)")}
  username: ${value("data.username", "base64(username)")}
  password: ${value("data.password", "base64(password)")}
  % endif
  % if cloud == "local":
  % endif
