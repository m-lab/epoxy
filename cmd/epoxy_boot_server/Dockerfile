# The standard google cloud-sdk container that "just works" within GCE.
FROM google/cloud-sdk

# Add the server to respond to HTTP requests at port 8080.
COPY epoxy_boot_server /
RUN chmod -R a+rx /epoxy_boot_server
ENTRYPOINT ["/epoxy_boot_server"]
