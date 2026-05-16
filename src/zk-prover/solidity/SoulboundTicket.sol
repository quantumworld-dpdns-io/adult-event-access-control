// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

// Soulbound Ticket (ERC-5192 compatible)
// Non-transferable NFT representing event access
// Requires ZK age proof verification before minting

interface IERC5192 {
    event Locked(uint256 tokenId);
    event Unlocked(uint256 tokenId);

    function locked(uint256 tokenId) external view returns (bool);
}

contract SoulboundTicket is IERC5192 {
    address public verifier;
    address public owner;
    uint256 public nextTokenId;

    struct Ticket {
        uint256 eventId;
        address attendee;
        string nullifierHash;
        bool ageVerified;
        bool checkedIn;
    }

    mapping(uint256 => Ticket) public tickets;
    mapping(address => uint256[]) public userTickets;
    mapping(string => bool) public usedNullifiers;

    event TicketMinted(uint256 tokenId, uint256 eventId, address attendee);
    event CheckedIn(uint256 tokenId, uint256 eventId);

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    constructor(address _verifier) {
        verifier = _verifier;
        owner = msg.sender;
    }

    function mint(
        uint256 eventId,
        address attendee,
        string calldata nullifierHash,
        bytes calldata ageProof
    ) external returns (uint256) {
        require(!usedNullifiers[nullifierHash], "Nullifier already used");

        // Verify ZK age proof
        (bool success, ) = verifier.call(ageProof);
        require(success, "Age proof verification failed");

        uint256 tokenId = nextTokenId++;
        tickets[tokenId] = Ticket({
            eventId: eventId,
            attendee: attendee,
            nullifierHash: nullifierHash,
            ageVerified: true,
            checkedIn: false
        });
        userTickets[attendee].push(tokenId);
        usedNullifiers[nullifierHash] = true;

        emit Locked(tokenId);
        emit TicketMinted(tokenId, eventId, attendee);
        return tokenId;
    }

    function checkIn(uint256 tokenId) external {
        Ticket storage t = tickets[tokenId];
        require(!t.checkedIn, "Already checked in");
        t.checkedIn = true;
        emit CheckedIn(tokenId, t.eventId);
    }

    function locked(uint256 tokenId) external pure returns (bool) {
        return true; // Always locked (soulbound)
    }

    function getTickets(address user) external view returns (uint256[] memory) {
        return userTickets[user];
    }
}
