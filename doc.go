// Copyright 2014 ShopeX. All rights reserved.

//
// Connect
//  c, err := teegon.NewClient("http://127.0.0.1:8080/api", "<key>", "<secret>")
//
// RPC
//
//  c.Get("shopex.query.appqueue", &map[string]interface{}{
//     "user_eid":"321312312",
//      "app_id": "test1",
//  })
//  ...
//  c.Post("shopex.query.appqueue", &map[string]interface{}{
//     "user_eid":"321312312",
//      "app_id": "test1",
//  })
//
//
package teegon
