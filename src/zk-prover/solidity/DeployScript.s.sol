// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Script.sol";
import "./AgeVerifier.sol";
import "./SoulboundTicket.sol";

contract DeployScript is Script {
    function run() external {
        uint256 deployerKey = vm.envUint("DEPLOYER_PRIVATE_KEY");

        vm.startBroadcast(deployerKey);

        // Deploy verifier with placeholder VK
        AgeVerifier verifier = new AgeVerifier(hex"00");

        // Deploy SBT contract linked to verifier
        SoulboundTicket sbt = new SoulboundTicket(address(verifier));

        vm.stopBroadcast();

        console.log("AgeVerifier deployed at:", address(verifier));
        console.log("SoulboundTicket deployed at:", address(sbt));
    }
}
