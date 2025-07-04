# banks_pdc_config_generator.py
import itertools
import json

# List of all MSPs
banks = [
    "AccessBankMSP", "GTBankMSP", "ZenithBankMSP", "FirstBankMSP",
    "CitiBankMSP", "EcoBankMSP", "FidelityBankMSP", "FirstCityMonumentBankMSP",
    "GlobusBankMSP", "KeystoneBankMSP", "OptimusBankMSP", "ParrallexBankMSP",
    "PolarisBankMSP", "PremiumTrustBankMSP", "ProvidusBankMSP", "StanbicIBTCBankMSP",
    "StandardCharteredBankMSP", "SterlingBankMSP", "SunTrustBankMSP", "TitanTrustBankMSP",
    "UnionBankMSP", "UBAMSP", "UnityBankMSP", "WemaBankMSP"
]

config = []

# BVN PDC
bvn_policy = "OR(" + ",".join(f"'{b}.member'" for b in banks + ["CentralBankPeerMSP"]) + ")"
config.append({
    "name": "col-BVN",
    "policy": bvn_policy,
    "memberOnlyRead": True,
    "memberOnlyWrite": True,
    "requiredPeerCount": 1,
    "maxPeerCount": len(banks) + 1,
    "blockToLive": 0,
    "endorsementPolicy": {"signaturePolicy": bvn_policy}
})

# Bilateral PDCs
for a, b in itertools.combinations(banks, 2):
    coll = f"col-{a}-{b}"
    policy = f"OR('{a}.member','{b}.member','CentralBankPeerMSP.member')"
    config.append({
        "name": coll,
        "policy": policy,
        "memberOnlyRead": True,
        "memberOnlyWrite": True,
        "requiredPeerCount": 1,
        "maxPeerCount": 3,
        "blockToLive": 0,
        "endorsementPolicy": {"signaturePolicy": policy}
    })

# Settlement collections
for b in banks:
    coll = f"col-settlement-{b}"
    policy = f"OR('{b}.member','CentralBankPeerMSP.member')"
    config.append({
        "name": coll,
        "policy": policy,
        "memberOnlyRead": False,
        "memberOnlyWrite": True,
        "requiredPeerCount": 1,
        "maxPeerCount": 2,
        "blockToLive": 0,
        "endorsementPolicy": {"signaturePolicy": policy}
    })

# Output JSON
print(json.dumps(config, indent=2))
