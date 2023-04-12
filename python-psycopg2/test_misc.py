import unittest
import psycopg2
from psycopg2.policies import ClusterAwareLoadBalancer as cs
import os
from psycopg2._psycopg import OperationalError

class TestMisc(unittest.TestCase):
    def setupfortest1(self):
        self.yb_path = os.getenv('YB_PATH')
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yugabyted start --ysql_port 5544')

    def setupfortest2(self):
        self.yb_path = os.getenv('YB_PATH')
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yb-ctl create --rf 3 ')

    def create_conns(self, url, numConns):
        conns = []
        for i in range(0, numConns):
            conn = psycopg2.connect(url)
            conns.append(conn)
        return conns
        
    def cleanup(self):
        os.system(self.yb_path+'/bin/yb-ctl destroy')
        os.system(self.yb_path+'/bin/yugabyted destroy')

    def cleanupconns(self, conns):
        for conn in conns:
            conn.close()
        conns.clear()

    def test_1(self):
        self.setupfortest1()
        def conn():
            conn = psycopg2.connect("user = yugabyte password=yugabyte host=localhost dbname=yugabyte load_balance=True")
        self.assertRaises(psycopg2.OperationalError,conn )
        self.cleanup()

    def test_2(self):
        self.setupfortest2()
        try:
            conn = psycopg2.connect("user = yugabyte password=yugabyte host=localhost dbname=yugabyte load_balance=True")
            conn.close()
        except OperationalError as e:
            self.fail("Test should have passed, but raised an OperationalError")
        finally:
            self.cleanup()

    # Test for connection via Connection URI Format

    def test_3(self):
        print("running test 3")
        self.setupfortest2()
        
        conns1 = self.create_conns("postgresql://yugabyte:yugabyte@127.0.0.1:5433/yugabyte?load_balance=True", 3)
        conns2 = self.create_conns("postgresql://yugabyte:yugabyte@127.0.0.1/yugabyte?load_balance=True", 3)
        conns3 = self.create_conns("postgresql://127.0.0.1:5433/yugabyte?user=yugabyte&load_balance=True",3)
        conns4 = self.create_conns("postgresql://127.0.0.1/yugabyte?user=yugabyte&load_balance=True", 3)
        obj = cs()
        numConnMap = obj.hostToNumConnMap
        for host,numconn in numConnMap.items():
            self.assertEqual(numconn, 4)
        self.cleanupconns(conns1)
        self.cleanupconns(conns2)
        self.cleanupconns(conns3)
        self.cleanupconns(conns4)
        self.cleanup()


if __name__ == '__main__':
    unittest.main()