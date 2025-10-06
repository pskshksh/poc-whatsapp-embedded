import React, { useEffect, useState } from "react";

const CONFIG = {
  APP_ID: "your_app_id_here",
  CONFIG_ID: "your_app_config_here",
  BACKEND_URL: "http://localhost:8081",
  REDIRECT_URI: "https://482e8d84cfc0.ngrok-free.app",
};

export default function App() {
  const [isSDKLoaded, setIsSDKLoaded] = useState(false);
  const [status, setStatus] = useState("idle");
  const [message, setMessage] = useState("");
  const [businessInfo, setBusinessInfo] = useState(null);
  const [nextSteps, setNextSteps] = useState([]);
  const [tokenInfo, setTokenInfo] = useState(null);
  const [wabaDetails, setWabaDetails] = useState(null);

  // URL Parameters Method (Only Working Method)
  const loadSDKWithURLParams = () => {
    console.log("Loading Facebook SDK with URL parameters method...");

    // Remove existing script if present
    const existingScript = document.getElementById("facebook-jssdk");
    if (existingScript) existingScript.remove();

    // Create script with all parameters in URL
    const script = document.createElement("script");
    script.id = "facebook-jssdk";
    script.async = true;
    script.defer = true;
    script.crossOrigin = "anonymous";
    script.src = `https://connect.facebook.net/en_US/sdk.js#xfbml=1&version=v19.0&appId=${CONFIG.APP_ID}&autoLogAppEvents=1`;

    script.onload = () => {
      console.log("Facebook SDK loaded with URL params");

      // Manual initialization for FB.login functionality
      setTimeout(() => {
        if (window.FB) {
          console.log(
            "Manually initializing FB SDK for login functionality...",
          );
          try {
            window.FB.init({
              appId: CONFIG.APP_ID,
              cookie: true,
              xfbml: true,
              version: "v19.0",
            });

            // Test that it's working
            window.FB.getLoginStatus(function (response) {
              console.log("Manual init successful, status:", response.status);
              setIsSDKLoaded(true);
            });
          } catch (error) {
            console.error("Manual init failed:", error);
            setMessage(`Manual init failed: ${error.message}`);
          }
        } else {
          console.error("FB object not available after load");
          setMessage("Facebook SDK loaded but FB object not available");
        }
      }, 1500);
    };

    script.onerror = (error) => {
      console.error("Failed to load Facebook SDK:", error);
      setMessage("Failed to load Facebook SDK");
    };

    document.head.appendChild(script);
  };

  // Initialize SDK on component mount
  useEffect(() => {
    // Ensure fb-root div exists
    if (!document.getElementById("fb-root")) {
      const fbRoot = document.createElement("div");
      fbRoot.id = "fb-root";
      document.body.insertBefore(fbRoot, document.body.firstChild);
      console.log("Added fb-root div");
    }

    // Load SDK using URL params method only
    loadSDKWithURLParams();

    // Cleanup timeout
    const timeout = setTimeout(() => {
      if (!isSDKLoaded) {
        console.warn("SDK loading timeout");
        setMessage("SDK loading timed out. Please refresh the page.");
      }
    }, 10000);

    return () => clearTimeout(timeout);
  }, [isSDKLoaded]);

  // Listen for Facebook messages and capture WABA details
  useEffect(() => {
    const messageListener = (event) => {
      if (
        !event.origin.endsWith(".facebook.com") &&
        event.origin !== "https://www.facebook.com" &&
        event.origin !== "https://business.facebook.com"
      ) {
        return;
      }

      try {
        const data =
          typeof event.data === "string" ? JSON.parse(event.data) : event.data;
        if (data?.type === "WA_EMBEDDED_SIGNUP") {
          console.log("WhatsApp Embedded Signup event:", data);

          if (data.event === "FINISH") {
            console.log(
              "Signup completed - capturing WABA details:",
              data.data,
            );
            // Capture WABA details from the message event
            if (data.data) {
              setWabaDetails({
                waba_id: data.data.waba_id,
                phone_number_id: data.data.phone_number_id,
                business_id: data.data.business_id,
              });
              console.log("Stored WABA details:", {
                waba_id: data.data.waba_id,
                phone_number_id: data.data.phone_number_id,
                business_id: data.data.business_id,
              });
            }
          } else if (data.event === "CANCEL") {
            setStatus("error");
            setMessage(
              `Signup cancelled: ${data.data?.current_step || "unknown"}`,
            );
          }
        }
      } catch (e) {
        // Ignore non-JSON messages
      }
    };

    window.addEventListener("message", messageListener);
    return () => window.removeEventListener("message", messageListener);
  }, []);

  const startSignup = () => {
    if (!isSDKLoaded || !window.FB) {
      setMessage("Facebook SDK not ready");
      return;
    }

    setStatus("loading");
    setMessage("Starting WhatsApp Business signup...");

    console.log("Starting FB.login with embedded signup");

    // Check FB status first
    window.FB.getLoginStatus(function (statusResponse) {
      console.log("Current FB status before login:", statusResponse);

      try {
        const loginConfig = {
          config_id: CONFIG.CONFIG_ID,
          response_type: "code",
          override_default_response_type: true,
          extras: {
            setup: {},
            featureType: "",
            sessionInfoVersion: "3",
          },
        };

        console.log("FB.login config:", loginConfig);

        window.FB.login((response) => {
          console.log("FB.login response:", response);

          if (response.error) {
            setStatus("error");
            setMessage(`Login error: ${response.error.message}`);
            return;
          }

          const code = response?.authResponse?.code;
          if (code) {
            console.log("Got authorization code");
            setupBusinessAccount(code);
          } else {
            setStatus("error");
            setMessage(`No code received. Status: ${response.status}`);
          }
        }, loginConfig);
      } catch (error) {
        console.error("FB.login error:", error);
        setStatus("error");
        setMessage(`FB.login failed: ${error.message}`);
      }
    });
  };

  const setupBusinessAccount = async (authCode) => {
    try {
      setMessage("Processing WhatsApp Business setup...");

      const requestPayload = {
        authorization_code: authCode,
        redirect_uri: CONFIG.REDIRECT_URI,
        // Include WABA details from message event if available
        ...(wabaDetails && {
          waba_id: wabaDetails.waba_id,
          phone_number_id: wabaDetails.phone_number_id,
          business_id: wabaDetails.business_id,
        }),
        client_info: {
          user_agent: navigator.userAgent,
          timestamp: new Date().toISOString(),
          waba_details_captured: !!wabaDetails,
        },
      };

      console.log("Sending to backend:", requestPayload);
      console.log("WABA Details:", wabaDetails);

      const res = await fetch(`${CONFIG.BACKEND_URL}/api/whatsapp/setup`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestPayload),
      });

      const result = await res.json();
      console.log("Backend response:", result);

      if (result.success) {
        setStatus("success");
        setMessage(result.message || "Setup completed!");
        setBusinessInfo(result.business_info);
        setNextSteps(result.next_steps || []);
        setTokenInfo(result.token_info || null);
      } else {
        throw new Error(result.error || "Setup failed");
      }
    } catch (err) {
      setStatus("error");
      setMessage(`Setup error: ${err.message}`);
    }
  };

  const reset = () => {
    setStatus("idle");
    setMessage("");
    setBusinessInfo(null);
    setNextSteps([]);
    setTokenInfo(null);
    setWabaDetails(null);
  };

  const testConnection = () => {
    if (window.FB) {
      window.FB.getLoginStatus((response) => {
        alert(`FB Status: ${response.status}\nSDK Working: ${!!window.FB}`);
      });
    } else {
      alert("FB object not available");
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        alert("Copied to clipboard!");
      })
      .catch(() => {
        alert("Failed to copy to clipboard");
      });
  };

  return (
    <div
      style={{
        maxWidth: 800,
        margin: "40px auto",
        fontFamily: "system-ui",
        padding: "20px",
      }}
    >
      {/* Ensure fb-root div exists */}
      <div id="fb-root"></div>

      <h1>WhatsApp Business — Embedded Signup</h1>
      <p style={{ color: "#666", marginBottom: "25px" }}>
        Complete WhatsApp Business Account setup with automatic token exchange
        and WABA configuration.
      </p>

      {/* Status Display */}
      <div
        style={{
          marginBottom: "25px",
          padding: "20px",
          backgroundColor: isSDKLoaded ? "#d4edda" : "#fff3cd",
          border: `2px solid ${isSDKLoaded ? "#c3e6cb" : "#ffeaa7"}`,
          borderRadius: "8px",
          fontSize: "14px",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            marginBottom: "10px",
          }}
        >
          <strong>Status:</strong>
          <span
            style={{
              marginLeft: "8px",
              padding: "4px 8px",
              borderRadius: "4px",
              backgroundColor: isSDKLoaded ? "#28a745" : "#ffc107",
              color: "white",
              fontSize: "12px",
              fontWeight: "600",
            }}
          >
            {isSDKLoaded ? "✅ SDK Ready" : "⏳ Loading..."}
          </span>
        </div>

        <div style={{ marginBottom: "8px" }}>
          <strong>Method:</strong> URL Parameters (Optimized)
        </div>
        <div style={{ marginBottom: "8px" }}>
          <strong>FB Object:</strong>{" "}
          {window.FB ? "✅ Available" : "❌ Not Available"}
        </div>
        <div style={{ marginBottom: "8px" }}>
          <strong>App ID:</strong> {CONFIG.APP_ID}
        </div>

        {wabaDetails && (
          <div
            style={{
              marginTop: "15px",
              padding: "12px",
              backgroundColor: "#e8f5e8",
              borderRadius: "6px",
              border: "1px solid #c3e6cb",
            }}
          >
            <strong style={{ color: "#155724" }}>
              🎯 WABA Details Captured:
            </strong>
            <div
              style={{
                marginTop: "8px",
                fontSize: "12px",
                fontFamily: "monospace",
              }}
            >
              <div>
                <strong>WABA ID:</strong> {wabaDetails.waba_id}
              </div>
              <div>
                <strong>Phone ID:</strong> {wabaDetails.phone_number_id}
              </div>
              <div>
                <strong>Business ID:</strong> {wabaDetails.business_id}
              </div>
            </div>
          </div>
        )}

        {message && (
          <div
            style={{
              marginTop: "12px",
              padding: "10px",
              backgroundColor: "#f8f9fa",
              borderRadius: "4px",
              fontSize: "13px",
            }}
          >
            <strong>Message:</strong> {message}
          </div>
        )}
      </div>

      {/* Test Connection Button */}
      {isSDKLoaded && (
        <div style={{ marginBottom: "20px" }}>
          <button
            onClick={testConnection}
            style={{
              padding: "8px 16px",
              backgroundColor: "#17a2b8",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
              fontSize: "13px",
            }}
          >
            🔍 Test FB Connection
          </button>
        </div>
      )}

      {status === "idle" && (
        <button
          onClick={startSignup}
          disabled={!isSDKLoaded}
          style={{
            padding: "18px 35px",
            fontSize: "18px",
            fontWeight: "600",
            backgroundColor: isSDKLoaded ? "#25d366" : "#ccc",
            color: "white",
            border: "none",
            borderRadius: "8px",
            cursor: isSDKLoaded ? "pointer" : "not-allowed",
            boxShadow: isSDKLoaded
              ? "0 4px 12px rgba(37, 211, 102, 0.3)"
              : "none",
            transition: "all 0.3s ease",
          }}
        >
          {isSDKLoaded ? "🚀 Connect WhatsApp Business" : "⏳ Loading SDK..."}
        </button>
      )}

      {status === "loading" && (
        <div style={{ textAlign: "center" }}>
          <h3 style={{ color: "#495057" }}>⏳ Setting up your account...</h3>
          <p style={{ fontSize: "16px", color: "#6c757d" }}>{message}</p>
          <div
            style={{
              width: "100%",
              height: "6px",
              backgroundColor: "#e9ecef",
              borderRadius: "3px",
              marginTop: "20px",
              overflow: "hidden",
            }}
          >
            <div
              style={{
                width: "60%",
                height: "100%",
                backgroundColor: "#25d366",
                borderRadius: "3px",
                animation: "pulse 1.5s ease-in-out infinite",
              }}
            ></div>
          </div>
        </div>
      )}

      {status === "success" && (
        <div>
          <h3 style={{ color: "#28a745", marginBottom: "20px" }}>
            ✅ Setup Complete!
          </h3>
          <p style={{ fontSize: "16px", marginBottom: "25px" }}>{message}</p>

          {/* Token Information Display */}
          {tokenInfo && (
            <div
              style={{
                marginBottom: "25px",
                padding: "20px",
                backgroundColor: "#f8f9fa",
                borderRadius: "8px",
                border: "2px solid #dee2e6",
              }}
            >
              <h4 style={{ marginBottom: "15px", color: "#495057" }}>
                🔑 Access Token Details
              </h4>
              <div style={{ fontSize: "14px", fontFamily: "monospace" }}>
                <div style={{ marginBottom: "10px" }}>
                  <strong>Token Type:</strong> {tokenInfo.token_type}
                </div>
                <div style={{ marginBottom: "10px" }}>
                  <strong>Token Length:</strong> {tokenInfo.access_token_length}{" "}
                  characters
                </div>
                <div style={{ marginBottom: "10px" }}>
                  <strong>Expires In:</strong> {tokenInfo.expires_in} seconds
                </div>
                <div style={{ marginBottom: "10px" }}>
                  <strong>Created At:</strong> {tokenInfo.token_created_at}
                </div>
                <div style={{ marginBottom: "15px" }}>
                  <strong>Preview:</strong> {tokenInfo.access_token_preview}
                </div>

                <div
                  style={{
                    backgroundColor: "#343a40",
                    color: "#f8f9fa",
                    padding: "15px",
                    borderRadius: "6px",
                    position: "relative",
                  }}
                >
                  <div style={{ marginBottom: "8px" }}>
                    <strong>Full Access Token:</strong>
                  </div>
                  <div
                    style={{
                      wordBreak: "break-all",
                      fontSize: "12px",
                      lineHeight: "1.4",
                      backgroundColor: "#495057",
                      padding: "10px",
                      borderRadius: "4px",
                    }}
                  >
                    {tokenInfo.full_access_token}
                  </div>
                  <button
                    onClick={() => copyToClipboard(tokenInfo.full_access_token)}
                    style={{
                      position: "absolute",
                      top: "10px",
                      right: "10px",
                      padding: "4px 8px",
                      backgroundColor: "#007bff",
                      color: "white",
                      border: "none",
                      borderRadius: "4px",
                      fontSize: "11px",
                      cursor: "pointer",
                    }}
                  >
                    📋 Copy
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Business Account Details */}
          {businessInfo && (
            <div
              style={{
                marginBottom: "25px",
                padding: "20px",
                backgroundColor: "#d4edda",
                borderRadius: "8px",
                border: "2px solid #c3e6cb",
              }}
            >
              <h4 style={{ marginBottom: "15px", color: "#155724" }}>
                🏢 Business Account Details
              </h4>
              <div style={{ marginBottom: "12px" }}>
                <strong>Business Name:</strong> {businessInfo.business_name}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>WABA ID:</strong> {businessInfo.waba_id}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>Account ID:</strong> {businessInfo.id}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>Phone Numbers:</strong>{" "}
                {businessInfo.phone_numbers?.length || 0}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>Webhooks:</strong>{" "}
                {businessInfo.webhooks_enabled ? "✅ Enabled" : "❌ Disabled"}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>Setup Status:</strong>{" "}
                {businessInfo.setup_complete ? "✅ Complete" : "⏳ In Progress"}
              </div>
              <div style={{ marginBottom: "12px" }}>
                <strong>Created:</strong>{" "}
                {new Date(businessInfo.created_at).toLocaleString()}
              </div>

              {/* Phone Numbers Details */}
              {businessInfo.phone_numbers?.map((phone, i) => (
                <div
                  key={phone.id || i}
                  style={{
                    marginTop: "20px",
                    paddingTop: "15px",
                    borderTop: "2px solid #c3e6cb",
                    backgroundColor: "#e8f5e8",
                    padding: "15px",
                    borderRadius: "6px",
                  }}
                >
                  <strong style={{ fontSize: "16px" }}>
                    📱 Phone Number {i + 1}
                  </strong>
                  <div style={{ marginTop: "8px", marginLeft: "15px" }}>
                    <div style={{ marginBottom: "6px" }}>
                      <strong>Phone Number:</strong> {phone.phone_number}
                    </div>
                    <div style={{ marginBottom: "6px" }}>
                      <strong>Phone ID:</strong> {phone.id}
                    </div>
                    <div style={{ marginBottom: "6px" }}>
                      <strong>Display Name:</strong> {phone.display_name}
                    </div>
                    <div style={{ marginBottom: "6px" }}>
                      <strong>Status:</strong> {phone.status}
                    </div>
                    <div style={{ marginBottom: "6px" }}>
                      <strong>Verified:</strong>{" "}
                      {phone.is_verified ? "✅ Yes" : "❌ No"}
                    </div>
                    <div>
                      <strong>Quality Rating:</strong>{" "}
                      {phone.quality_rating || "Unknown"}
                    </div>
                  </div>
                </div>
              ))}

              {/* Metadata */}
              {businessInfo.metadata && (
                <div
                  style={{
                    marginTop: "20px",
                    paddingTop: "15px",
                    borderTop: "2px solid #c3e6cb",
                  }}
                >
                  <strong>📋 Additional Metadata:</strong>
                  <div
                    style={{
                      marginTop: "8px",
                      fontSize: "13px",
                      fontFamily: "monospace",
                      backgroundColor: "#f8f9fa",
                      padding: "10px",
                      borderRadius: "4px",
                    }}
                  >
                    {JSON.stringify(businessInfo.metadata, null, 2)}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Next Steps */}
          {nextSteps.length > 0 && (
            <div style={{ marginBottom: "25px" }}>
              <h4 style={{ marginBottom: "15px" }}>🎯 Next Steps:</h4>
              <ul style={{ paddingLeft: "25px", lineHeight: "1.6" }}>
                {nextSteps.map((step, i) => (
                  <li key={i} style={{ marginBottom: "8px", fontSize: "15px" }}>
                    {step}
                  </li>
                ))}
              </ul>
            </div>
          )}

          <button
            onClick={reset}
            style={{
              padding: "12px 25px",
              backgroundColor: "#6c757d",
              color: "white",
              border: "none",
              borderRadius: "6px",
              cursor: "pointer",
              fontSize: "14px",
            }}
          >
            🔄 Setup Another Account
          </button>
        </div>
      )}

      {status === "error" && (
        <div>
          <h3 style={{ color: "#dc3545" }}>❌ Setup Failed</h3>
          <div
            style={{
              padding: "20px",
              backgroundColor: "#f8d7da",
              color: "#721c24",
              borderRadius: "8px",
              marginBottom: "20px",
              border: "2px solid #f5c6cb",
            }}
          >
            <strong>Error Details:</strong>
            <div style={{ marginTop: "10px", fontSize: "14px" }}>{message}</div>
          </div>
          <button
            onClick={reset}
            style={{
              padding: "12px 25px",
              backgroundColor: "#dc3545",
              color: "white",
              border: "none",
              borderRadius: "6px",
              cursor: "pointer",
              fontSize: "14px",
            }}
          >
            🔄 Try Again
          </button>
        </div>
      )}

      <style jsx>{`
        @keyframes pulse {
          0%,
          100% {
            opacity: 1;
          }
          50% {
            opacity: 0.5;
          }
        }
      `}</style>
    </div>
  );
}
