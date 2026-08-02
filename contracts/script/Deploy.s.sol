// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

import { Script } from "forge-std/Script.sol";
import { console2 } from "forge-std/console2.sol";
import { MyferenceMarket } from "../src/MyferenceMarket.sol";

contract DeployMyferenceMarket is Script {
    function run() external returns (MyferenceMarket market) {
        uint256 deployerKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        address owner = vm.envAddress("MYFERENCE_OWNER");
        address feeRecipient = vm.envAddress("MYFERENCE_FEE_RECIPIENT");
        address settlementSigner = vm.envAddress("MYFERENCE_SETTLEMENT_SIGNER");
        uint256 minimumBond = vm.envUint("MYFERENCE_MINIMUM_BOND_WEI");
        uint64 bondExitDelay = uint64(vm.envUint("MYFERENCE_BOND_EXIT_DELAY_SECONDS"));
        uint64 feeDelay = uint64(vm.envUint("MYFERENCE_FEE_DELAY_SECONDS"));

        vm.startBroadcast(deployerKey);
        market = new MyferenceMarket(
            owner, feeRecipient, settlementSigner, minimumBond, bondExitDelay, feeDelay
        );
        vm.stopBroadcast();

        console2.log("MyferenceMarket", address(market));
        console2.log("chainId", block.chainid);
    }
}
