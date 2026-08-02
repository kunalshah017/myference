// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

import { StdInvariant } from "forge-std/StdInvariant.sol";
import { Test } from "forge-std/Test.sol";
import { MyferenceMarket } from "../src/MyferenceMarket.sol";

contract MarketHandler is Test {
    MyferenceMarket public immutable market;
    address public immutable customer;
    address public immutable provider;
    address public immutable settlementSigner;

    uint256 internal constant CUSTOMER_KEY = 0xCAFE;
    uint256 internal constant PROVIDER_KEY = 0xA11CE;
    uint256 internal constant SETTLEMENT_KEY = 0xB0B;
    bytes32 internal constant OFFER_ID = keccak256("invariant-offer");
    bytes32 internal constant MODEL_HASH = keccak256("invariant-model");
    bytes32 internal constant CAPABILITY_HASH = keccak256("chat-completions");

    bytes32[] public sessionIds;
    bytes32[] public settledRequestIds;
    uint64 internal nextNonce = 1;

    constructor(MyferenceMarket market_) {
        market = market_;
        customer = vm.addr(CUSTOMER_KEY);
        provider = vm.addr(PROVIDER_KEY);
        settlementSigner = vm.addr(SETTLEMENT_KEY);
        vm.deal(provider, 10 ether);
        vm.startPrank(provider);
        market.depositBond{ value: 10 ether }();
        market.publishOffer(OFFER_ID, MODEL_HASH, CAPABILITY_HASH, 1, 0, 0);
        vm.stopPrank();
    }

    function deposit(uint96 seed) external {
        uint256 amount = bound(uint256(seed), 1, 1 ether);
        vm.deal(customer, customer.balance + amount);
        vm.prank(customer);
        market.deposit{ value: amount }();
    }

    function withdraw(uint96 seed) external {
        uint256 balance = market.customerBalances(customer);
        if (balance == 0) return;
        vm.prank(customer);
        market.requestWithdrawal(bound(uint256(seed), 1, balance));
    }

    function openSession(uint96 seed) external {
        uint256 balance = market.customerBalances(customer);
        if (balance == 0) return;
        bytes32 sessionId = keccak256(abi.encode("session", sessionIds.length));
        uint256 allowance = bound(uint256(seed), 1, balance);
        vm.prank(customer);
        market.openSession(sessionId, allowance, uint64(block.timestamp + 30 days));
        sessionIds.push(sessionId);
    }

    function settle(uint256 seed) external {
        if (sessionIds.length == 0) return;
        bytes32 sessionId = sessionIds[seed % sessionIds.length];
        (
            address sessionCustomer,
            uint256 allowance,
            uint256 spent,
            uint64 expiresAt,,
            bool finalized
        ) = market.sessions(sessionId);
        if (finalized || allowance - spent == 0) return;
        bytes32 requestId = keccak256(abi.encode("request", nextNonce));
        MyferenceMarket.Receipt memory receipt = MyferenceMarket.Receipt({
            requestId: requestId,
            sessionId: sessionId,
            customer: sessionCustomer,
            provider: provider,
            settlementSigner: settlementSigner,
            offerId: OFFER_ID,
            priceVersion: 1,
            modelHash: MODEL_HASH,
            capabilityHash: CAPABILITY_HASH,
            inputTokens: 1,
            outputTokens: 0,
            computeMilliseconds: 0,
            maximumCharge: 1,
            totalCharge: 1,
            feeBasisPoints: 500,
            feeVersion: 1,
            status: 1,
            completedAt: uint64(block.timestamp > expiresAt ? expiresAt : block.timestamp),
            inputHash: keccak256("input"),
            outputHash: keccak256("output"),
            nonce: nextNonce
        });
        bytes32 digest = market.hashReceipt(receipt);
        (uint8 pv, bytes32 pr, bytes32 ps) = vm.sign(PROVIDER_KEY, digest);
        (uint8 sv, bytes32 sr, bytes32 ss) = vm.sign(SETTLEMENT_KEY, digest);
        market.settleReceipt(receipt, abi.encodePacked(pr, ps, pv), abi.encodePacked(sr, ss, sv));
        settledRequestIds.push(requestId);
        ++nextNonce;
    }

    function sessionCount() external view returns (uint256) {
        return sessionIds.length;
    }

    function settledRequestCount() external view returns (uint256) {
        return settledRequestIds.length;
    }
}

contract MyferenceMarketInvariantTest is StdInvariant, Test {
    MyferenceMarket internal market;
    MarketHandler internal handler;

    function setUp() public {
        market = new MyferenceMarket(
            address(this), address(0xFEE), vm.addr(0xB0B), 10 ether, 7 days, 2 days
        );
        handler = new MarketHandler(market);
        targetContract(address(handler));
    }

    function invariantMonIsConserved() public view {
        assertEq(
            address(market).balance,
            market.totalCustomerBalances() + market.totalProviderBonds()
                + market.totalLockedSessions() + market.totalClaimable()
        );
    }

    function invariantSessionsNeverExceedAllowance() public view {
        for (uint256 i; i < handler.sessionCount(); ++i) {
            (, uint256 allowance, uint256 spent,,,) = market.sessions(handler.sessionIds(i));
            assertLe(spent, allowance);
        }
    }

    function invariantSettledRequestsStaySettled() public view {
        for (uint256 i; i < handler.settledRequestCount(); ++i) {
            assertTrue(market.settledRequests(handler.settledRequestIds(i)));
        }
    }

    function invariantFeeNeverExceedsCeiling() public view {
        assertLe(market.feeBasisPoints(), market.MAXIMUM_FEE_BASIS_POINTS());
    }
}
