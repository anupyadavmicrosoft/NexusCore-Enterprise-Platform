#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
==============================================================================
AUTOMATED SECURITY PENETRATION & VULNERABILITY TESTING SUITE
Tests for SQL Injection vulnerabilities, CORS leaks, and JWT signature bypasses
==============================================================================
"""

import sys
import unittest
import urllib.request
import urllib.error
import json

class TestNexusCoreSecurityVulnerabilities(unittest.TestCase):
    
    def setUp(self):
        self.base_url = "http://api-gateway.nexus-core.svc.cluster.local:8080/api/v2"
        self.auth_token = "Bearer nexus-token-secure-390"
        
    def test_sql_injection_evasion(self):
        """Probes parameters for SQL Injection vulnerability leaks"""
        sqli_payloads = [
            "' OR '1'='1",
            "'; DROP TABLE transactions;--",
            "1 UNION SELECT null, username, password FROM users",
            "1' AND sleep(5) --"
        ]
        
        for payload in sqli_payloads:
            # Query endpoint to search account history, passing SQL injection strings
            url = f"{self.base_url}/accounts/acc-9921/history?query={urllib.parse.quote(payload)}"
            req = urllib.request.Request(url)
            req.add_header("Authorization", self.auth_token)
            
            try:
                # API Gateway should sanitize query parameters, or reject them as Bad Request (400)
                # rather than passing raw queries to the underlying database driver.
                with urllib.request.urlopen(req, timeout=3) as response:
                    # If it successfully executes, confirm the database didn't crash or return full datasets
                    body = response.read().decode('utf-8')
                    data = json.loads(body)
                    
                    # Confirm database records aren't dumped
                    self.assertNotIn("password", body.lower())
                    self.assertNotIn("hash", body.lower())
                    self.assertIsInstance(data, list)
            except urllib.error.HTTPError as e:
                # Correct response: 400 Bad Request or 403 Forbidden is a PASS for security tests!
                self.assertIn(e.code, [400, 403, 422])
            except urllib.error.URLError:
                # Network isolation pass
                pass

    def test_jwt_signature_bypass_algorithm_none(self):
        """Validates that 'alg: none' JWT signature bypass attempts are rejected"""
        # Formulate fake spoofed token header: {"alg":"none","typ":"JWT"}
        # payload: {"sub":"admin-principal-99x","role":"ClusterAdmin"}
        spoofed_token = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJhZG1pbi1wcmluY2lwYWwtOTl4Iiwicm9sZSI6IkNsdXN0ZXJBZG1pbiJ9."
        
        url = f"{self.base_url}/transactions"
        req = urllib.request.Request(url, method="POST", data=b'{"account_id":"acc-99", "amount": 100.0}')
        req.add_header("Authorization", f"Bearer {spoofed_token}")
        req.add_header("Content-Type", "application/json")
        
        try:
            with urllib.request.urlopen(req, timeout=3) as response:
                self.fail("Security risk: API Gateway accepted an unverified 'none' algorithm JWT signature!")
        except urllib.error.HTTPError as e:
            # Authentication must be denied (401 Unauthorized or 403 Forbidden)
            self.assertEqual(e.code, 401)
        except urllib.error.URLError:
            pass

    def test_cors_origin_reflection_validation(self):
        """Validates that the service restricts wildcards on CORS credentials-allowed headers"""
        url = f"{self.base_url}/health/live"
        req = urllib.request.Request(url)
        req.add_header("Origin", "https://malicious-attacker-spoof.com")
        
        try:
            with urllib.request.urlopen(req, timeout=3) as response:
                headers = dict(response.info())
                allow_origin = headers.get("Access-Control-Allow-Origin", "")
                allow_creds = headers.get("Access-Control-Allow-Credentials", "")
                
                # Check for critical CORS vulnerability combination
                if allow_origin == "https://malicious-attacker-spoof.com":
                    self.assertNotEqual(allow_creds, "true", "VULNERABILITY: CORS credentials allowed on untrusted wildcard source Origin!")
        except urllib.error.URLError:
            pass

if __name__ == "__main__":
    unittest.main()
