import React from 'react';

const App: React.FC = () => {
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-green-600 mb-4">RideShare Germany</h1>
        <p className="text-gray-600">Web Portal & Landing Page Scaffolded</p>
        <button className="mt-6 px-6 py-2 bg-green-600 text-white rounded-full hover:bg-green-700 transition">
          Sign Up as Driver
        </button>
      </div>
    </div>
  );
};

export default App;
