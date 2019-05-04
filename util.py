from aiohttp import web
from bitstring import BitArray


class Util:

    def __init__(self, banano_mode : bool):
        self.banano_mode = banano_mode
        self.raw_per_nano = 10**29 if banano_mode else 10**30

    def get_request_ip(self, r : web.Request) -> str:
        host = r.headers.get('X-FORWARDED-FOR',None)
        if host is None:
            peername = r.transport.get_extra_info('peername')
            if peername is not None:
                host, _ = peername
        return host

    def address_decode(self, address : str) -> str:
        """Given a string containing an XRB/NANO/BAN address, confirm validity and provide resulting hex address"""
        if (address[:4] == 'xrb_' or address[:5] == 'nano_' and not self.banano_mode) or (address[:4] == 'ban_' and self.banano_mode):
            account_map = "13456789abcdefghijkmnopqrstuwxyz"  # each index = binary value, account_lookup[0] == '1'
            account_lookup = {}
            for i in range(0, 32):  # populate lookup index with prebuilt bitarrays ready to append
                account_lookup[account_map[i]] = BitArray(uint=i, length=5)
            data = address.split('_')[1]
            acrop_key = data[:-8]  # we want everything after 'xrb_' or 'nano_' but before the 8-char checksum
            acrop_check = data[-8:]  # extract checksum

            # convert base-32 (5-bit) values to byte string by appending each 5-bit value to the bitstring,
            # essentially bitshifting << 5 and then adding the 5-bit value.
            number_l = BitArray()
            for x in range(0, len(acrop_key)):
                number_l.append(account_lookup[acrop_key[x]])

            number_l = number_l[4:]  # reduce from 260 to 256 bit (upper 4 bits are never used as account is a uint256)
            check_l = BitArray()

            for x in range(0, len(acrop_check)):
                check_l.append(account_lookup[acrop_check[x]])
            check_l.byteswap()  # reverse byte order to match hashing format
            result = number_l.hex.upper()
            return result

        return False

    def pubkey(self, address : str) -> str:
        """Account to public key"""
        account_map = "13456789abcdefghijkmnopqrstuwxyz"
        account_lookup = {}
        for i in range(0,32): #make a lookup table
            account_lookup[account_map[i]] = BitArray(uint=i,length=5)
        acrop_key = address[-60:-8] #leave out prefix and checksum
        number_l = BitArray()                                    
        for x in range(0, len(acrop_key)):    
            number_l.append(account_lookup[acrop_key[x]])        
        number_l = number_l[4:] # reduce from 260 to 256 bit
        result = number_l.hex.upper()
        return result

    def minimalNumber(self, x):
        if type(x) is str:
            if x == '':
                x = 0
        f = float(x)
        if f.is_integer():
            return int(f)
        else:
            return round(f, 2 if self.banano_mode else 6)

    def raw_to_nano(self, raw_amt : int):
        nano_amt = raw_amt / self.raw_per_nano
        # Format to have optional decimals
        return self.minimalNumber(nano_amt)

    def nano_to_raw(self, nano_amt):
        if not self.banano_mode:
            expanded = float(nano_amt) * 1000000
            return int(expanded) * (10 ** 24)
        else:
            expanded = float(nano_amt) * 100
            return int(expanded) * (10 ** 27)
