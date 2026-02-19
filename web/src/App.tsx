import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import LandingPage from './pages/LandingPage';

// Placeholder components for future implementation
const RiderSignup = () => <div className="p-20 text-center">Rider Signup Coming Soon</div>;
const DriverSignup = () => <div className="p-20 text-center">Driver Signup Coming Soon</div>;

function App() {
  return (
    <Router>
      <div className="App">
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/signup/rider" element={<RiderSignup />} />
          <Route path="/signup/driver" element={<DriverSignup />} />
          <Route path="*" element={<LandingPage />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
