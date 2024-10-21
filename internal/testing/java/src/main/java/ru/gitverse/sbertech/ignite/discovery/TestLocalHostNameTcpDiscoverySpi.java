package ru.gitverse.sbertech.ignite.discovery;

import org.apache.ignite.spi.discovery.tcp.TcpDiscoverySpi;

class TestLocalHostNameTcpDiscoverySpi extends TcpDiscoverySpi {
    @Override protected void initLocalNode(int srvPort, boolean addExtAddrAttr) {
        super.initLocalNode(srvPort, addExtAddrAttr);

        locNode.hostNames().add("localhost");
    }
}